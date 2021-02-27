package post

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type Order int

const (
	Ascending Order = iota
	Descending
)

type Post struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

type SearchFilter struct {
	Name   string
	Author string
	Order  Order
}

type postService struct {
	db     *redis.Client
	logger *zap.SugaredLogger
}

func NewService(db *redis.Client, logger *zap.SugaredLogger) Service {
	return postService{db, logger}
}

func (ps postService) Create(ctx context.Context, post *Post) (int64, error) {
	id, err := ps.db.Incr(ctx, "next_post_id").Result()
	if err != nil {
		return 0, err
	}

	_, err = ps.db.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, fmt.Sprintf("post:%v", id), "name", post.Name, "author", post.Author, "created_at", post.CreatedAt.Unix())
		pipe.SAdd(ctx, fmt.Sprintf("names:%v", post.Name), id)
		pipe.SAdd(ctx, fmt.Sprintf("authors:%v", post.Author), id)

		return nil
	})

	if err != nil {
		return 0, err
	}

	return id, nil
}

func (ps postService) FindOne(ctx context.Context, id int64) (*Post, error) {
	key := fmt.Sprintf("post:%v", id)

	exists, err := ps.db.Exists(ctx, key).Result()
	if exists == 0 {
		return nil, errors.New("post not found")
	}

	res, err := ps.db.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	return toPost(id, res)
}

func (ps postService) FindMany(ctx context.Context, filter *SearchFilter) ([]*Post, error) {
	var ids []int64
	switch {
	case filter.Author != "" && filter.Name != "":
		err := ps.db.SInter(ctx, "authors:"+filter.Author, "names:"+filter.Name).ScanSlice(&ids)
		if err != nil {
			return nil, err
		}
	case filter.Author != "" && filter.Name == "":
		err := ps.db.SMembers(ctx, "authors:"+filter.Author).ScanSlice(&ids)
		if err != nil {
			return nil, err
		}
	case filter.Author == "" && filter.Name != "":
		err := ps.db.SMembers(ctx, "names:"+filter.Name).ScanSlice(&ids)
		if err != nil {
			return nil, err
		}
	default:
		var (
			keys   []string
			cursor uint64
			err    error
		)

		for {
			keys, cursor, err = ps.db.Scan(ctx, cursor, "post:*", 10).Result()
			if err != nil {
				return nil, err
			}

			for _, key := range keys {
				strID := strings.TrimPrefix(key, "post:")

				id, err := strconv.ParseInt(strID, 10, 64)
				if err != nil {
					ps.logger.Warnf("failed to parse post ID: %v, error: %v", strID, err)
					continue
				}

				ids = append(ids, id)
			}

			if cursor == 0 {
				break
			}
		}
	}

	rawPosts := make(map[int64]*redis.StringStringMapCmd)
	_, err := ps.db.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, id := range ids {
			rawPosts[id] = pipe.HGetAll(ctx, fmt.Sprintf("post:%v", id))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	posts := make([]*Post, 0, len(rawPosts))
	for id, rp := range rawPosts {
		m, err := rp.Result()
		if err != nil {
			ps.logger.Errorf("Error while fetching: %v", err)
			continue
		}

		post, err := toPost(id, m)
		if err != nil {
			ps.logger.Errorf("Error while fetching: %v", err)
			continue
		}
		posts = append(posts, post)
	}

	switch filter.Order {
	case Ascending:
		sort.Slice(posts, func(i, j int) bool {
			return posts[i].CreatedAt.Before(posts[j].CreatedAt)
		})
	case Descending:
		sort.Slice(posts, func(i, j int) bool {
			return posts[i].CreatedAt.After(posts[j].CreatedAt)
		})
	}

	return posts, nil
}

func (ps postService) Remove(ctx context.Context, id int64) (bool, error) {
	post, err := ps.FindOne(ctx, id)
	if err != nil {
		return false, err
	}

	_, err = ps.db.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, fmt.Sprintf("post:%v", id))
		pipe.SRem(ctx, fmt.Sprintf("names:%v", post.Name), id)
		pipe.SRem(ctx, fmt.Sprintf("authors:%v", post.Author), id)

		return nil
	})

	if err != nil {
		return false, err
	}

	return true, nil
}

func (ps postService) Count(ctx context.Context) (map[string]int, error) {
	var (
		authors []string
		cursor  uint64
		err     error
	)

	for {
		var keys []string
		keys, cursor, err = ps.db.Scan(ctx, cursor, "authors:*", 10).Result()
		if err != nil {
			return nil, err
		}

		for _, k := range keys {
			authors = append(authors, strings.TrimPrefix(k, "authors:"))
		}

		if cursor == 0 {
			break
		}
	}

	authorRes := make(map[string]*redis.StringSliceCmd)
	_, err = ps.db.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, author := range authors {
			authorRes[author] = pipe.SMembers(ctx, fmt.Sprintf("authors:%v", author))
		}

		return nil
	})

	postsCount := make(map[string]int)
	for author, res := range authorRes {
		slice, err := res.Result()
		if err != nil {
			ps.logger.Errorf("Error while counting: %v", err)
			continue
		}

		postsCount[author] = len(slice)
	}

	return postsCount, nil
}

func (ps postService) Logger() *zap.SugaredLogger {
	return ps.logger
}

func toPost(id int64, res map[string]string) (*Post, error) {
	unix, err := strconv.ParseInt(res["created_at"], 0, 64)
	if err != nil {
		return nil, err
	}

	return &Post{
		ID:        id,
		Name:      res["name"],
		Author:    res["author"],
		CreatedAt: time.Unix(unix, 0),
	}, nil
}
