package post

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestPostServiceLogger(t *testing.T) {
	logger := zap.NewExample().Sugar()
	ps := postService{nil, logger}

	assert.Equal(t, ps.Logger(), logger)
}

func TestToPost(t *testing.T) {
	tests := []map[string]string{
		{
			"name":       "test 1",
			"author":     "vt",
			"created_at": "12345678",
		},
		{
			"name":       "test 2",
			"author":     "robot",
			"created_at": "0.1",
		},
	}

	post, err := toPost(1, tests[0])
	if assert.NoError(t, err) {
		unix, _ := strconv.ParseInt(tests[0]["created_at"], 10, 64)
		ts := time.Unix(unix, 0)

		assert.Equal(t, post.Author, tests[0]["author"])
		assert.Equal(t, post.Name, tests[0]["name"])
		assert.Equal(t, post.CreatedAt, ts)
	}

	post, err = toPost(2, tests[1])
	if assert.Error(t, err) {
		var numErr *strconv.NumError
		assert.ErrorAs(t, err, &numErr)
	}
}

func TestCount(t *testing.T) {
	client, mock := redismock.NewClientMock()
	logger := zap.NewExample().Sugar()
	ps := NewService(client, logger)

	mock.ExpectScan(0, "authors:*", 10).SetVal([]string{"vt", "robot"}, 0)
	mock.ExpectSMembers("authors:vt").SetVal([]string{"1", "2"})
	mock.ExpectSMembers("authors:robot").SetVal([]string{"3", "4", "5"})

	count, err := ps.Count(context.Background())
	if assert.NoError(t, err) {
		assert.Equal(t, 2, count["vt"])
		assert.Equal(t, 3, count["robot"])
	}
}

func TestCreate(t *testing.T) {
	client, mock := redismock.NewClientMock()
	logger := zap.NewExample().Sugar()
	ps := NewService(client, logger)

	tests := []struct {
		expected int64
		err      bool
		post     *Post
		mock     func()
	}{
		{
			expected: 1,
			err:      false,
			post:     &Post{Name: "t", Author: "t", CreatedAt: time.Unix(1, 0)},
			mock: func() {
				mock.ExpectIncr("next_post_id").SetVal(1)
				mock.ExpectHSet("post:1", "name", "t", "author", "t", "created_at", int64(1)).SetVal(1)
				mock.ExpectSAdd("names:t", int64(1)).SetVal(1)
				mock.ExpectSAdd("authors:t", int64(1)).SetVal(1)
			},
		},
		{
			expected: 0,
			err:      true,
			post:     &Post{Name: "t", Author: "t", CreatedAt: time.Unix(1, 0)},
			mock: func() {
				mock.ExpectIncr("next_post_id").SetVal(1)
				mock.ExpectHSet("post:1", "name", "t", "author", "t", "created_at", int64(1)).SetVal(1)
				mock.ExpectSAdd("names:t", int64(1)).SetErr(errors.New("fail"))
				mock.ExpectSAdd("authors:t", int64(1)).SetVal(1)
			},
		},
	}

	for _, test := range tests {
		test.mock()

		id, err := ps.Create(context.Background(), test.post)
		if test.err {
			assert.Error(t, err)
		} else {
			if assert.NoError(t, err) {
				assert.Equal(t, test.expected, id)
			}
		}

		mock.ClearExpect()
	}
}

func TestFindOne(t *testing.T) {
	client, mock := redismock.NewClientMock()
	logger := zap.NewExample().Sugar()
	ps := NewService(client, logger)

	tests := []struct {
		expected *Post
		id       int64
		err      bool
		mock     func()
	}{
		{
			expected: &Post{
				ID:        1,
				Name:      "test 1",
				Author:    "vt",
				CreatedAt: time.Unix(1, 0),
			},
			err: false,
			id:  1,
			mock: func() {
				mock.ExpectExists("post:1").SetVal(1)
				mock.ExpectHGetAll("post:1").SetVal(map[string]string{
					"name":       "test 1",
					"author":     "vt",
					"created_at": "1",
				})
			},
		},
		{
			expected: nil,
			err:      true,
			id:       2,
			mock: func() {
				mock.ExpectExists("post:2").SetVal(0)
			},
		},
		{
			expected: nil,
			err:      true,
			id:       3,
			mock: func() {
				mock.ExpectExists("post:3").SetVal(1)
				mock.ExpectHGetAll("post:3").SetErr(errors.New("fail"))
			},
		},
	}

	for _, test := range tests {
		test.mock()

		post, err := ps.FindOne(context.Background(), test.id)
		if test.err {
			assert.Error(t, err)
		} else {
			if assert.NoError(t, err) {
				assert.Equal(t, test.expected, post)
			}
		}

		mock.ClearExpect()
	}
}

func TestRemove(t *testing.T) {
	client, mock := redismock.NewClientMock()
	logger := zap.NewExample().Sugar()
	ps := NewService(client, logger)

	tests := []struct {
		expected bool
		id       int64
		err      bool
		mock     func()
	}{
		{
			expected: true,
			err:      false,
			id:       1,
			mock: func() {
				mock.ExpectExists("post:1").SetVal(1)
				mock.ExpectHGetAll("post:1").SetVal(map[string]string{
					"name":       "test 1",
					"author":     "vt",
					"created_at": "1",
				})

				mock.ExpectDel("post:1").SetVal(1)
				mock.ExpectSRem("names:test 1", int64(1)).SetVal(1)
				mock.ExpectSRem("authors:vt", int64(1)).SetVal(1)
			},
		},
		{
			expected: false,
			err:      true,
			id:       1,
			mock: func() {
				mock.ExpectExists("post:1").SetVal(1)
				mock.ExpectHGetAll("post:1").SetVal(map[string]string{
					"name":       "test 1",
					"author":     "vt",
					"created_at": "1",
				})

				mock.ExpectDel("post:1").SetErr(errors.New("fail"))
			},
		},
		{
			expected: false,
			err:      true,
			id:       2,
			mock: func() {
				mock.ExpectExists("post:2").SetVal(0)
			},
		},
	}

	for _, test := range tests {
		test.mock()

		removed, err := ps.Remove(context.Background(), test.id)
		if test.err {
			assert.Error(t, err)
		} else {
			if assert.NoError(t, err) {
				assert.Equal(t, test.expected, removed)
			}
		}

		mock.ClearExpect()
	}
}

func TestFindMany(t *testing.T) {
	client, mock := redismock.NewClientMock()
	logger := zap.NewExample().Sugar()
	ps := NewService(client, logger)

	tests := []struct {
		name     string
		expected []*Post
		filters  *SearchFilter
		err      bool
		mock     func()
	}{
		{
			name: "Filter by names. Success.",
			expected: []*Post{
				{1, "found", "vt", time.Unix(2, 0)},
				{2, "found", "vt", time.Unix(1, 0)},
			},
			err:     false,
			filters: &SearchFilter{Name: "found"},
			mock: func() {
				mock.ExpectSMembers("names:found").SetVal([]string{"1", "2"})
				mock.ExpectHGetAll("post:1").SetVal(map[string]string{
					"name":       "found",
					"author":     "vt",
					"created_at": "2",
				})
				mock.ExpectHGetAll("post:2").SetVal(map[string]string{
					"name":       "found",
					"author":     "vt",
					"created_at": "1",
				})
			},
		},
		{
			name: "Filter by authors. Success.",
			expected: []*Post{
				{2, "test1", "vt", time.Unix(2, 0)},
				{3, "test2", "vt", time.Unix(1, 0)},
			},
			err:     false,
			filters: &SearchFilter{Author: "vt"},
			mock: func() {
				mock.ExpectSMembers("authors:vt").SetVal([]string{"2", "3"})
				mock.ExpectHGetAll("post:2").SetVal(map[string]string{
					"name":       "test1",
					"author":     "vt",
					"created_at": "2",
				})
				mock.ExpectHGetAll("post:3").SetVal(map[string]string{
					"name":       "test2",
					"author":     "vt",
					"created_at": "1",
				})
			},
		},
		{
			name: "Filter by both. Success",
			expected: []*Post{
				{3, "found", "vt", time.Unix(2, 0)},
				{4, "found", "vt", time.Unix(1, 0)},
			},
			err:     false,
			filters: &SearchFilter{Name: "found", Author: "vt"},
			mock: func() {
				mock.ExpectSInter("authors:vt", "names:found").SetVal([]string{"3", "4"})
				mock.ExpectHGetAll("post:3").SetVal(map[string]string{
					"name":       "found",
					"author":     "vt",
					"created_at": "2",
				})
				mock.ExpectHGetAll("post:4").SetVal(map[string]string{
					"name":       "found",
					"author":     "vt",
					"created_at": "1",
				})
			},
		},
		{
			expected: []*Post{
				{1, "test1", "vt", time.Unix(4, 0)},
				{2, "test2", "vt", time.Unix(3, 0)},
				{3, "found", "vt", time.Unix(2, 0)},
				{4, "found", "vt", time.Unix(1, 0)},
			},
			err:     false,
			filters: &SearchFilter{},
			mock: func() {
				mock.ExpectScan(0, "post:*", 10).SetVal([]string{"post:1", "post:2"}, 30)
				mock.ExpectScan(30, "post:*", 10).SetVal([]string{"post:3", "post:4"}, 0)

				mock.ExpectHGetAll("post:1").SetVal(map[string]string{
					"name":       "test1",
					"author":     "vt",
					"created_at": "4",
				})
				mock.ExpectHGetAll("post:2").SetVal(map[string]string{
					"name":       "test2",
					"author":     "vt",
					"created_at": "3",
				})
				mock.ExpectHGetAll("post:3").SetVal(map[string]string{
					"name":       "found",
					"author":     "vt",
					"created_at": "2",
				})
				mock.ExpectHGetAll("post:4").SetVal(map[string]string{
					"name":       "found",
					"author":     "vt",
					"created_at": "1",
				})
			},
		},
		{
			name: "Filter by names. Ascending order.",
			expected: []*Post{
				{1, "found", "vt", time.Unix(1, 0)},
				{2, "found", "vt", time.Unix(2, 0)},
			},
			err:     false,
			filters: &SearchFilter{Name: "found", Order: Ascending},
			mock: func() {
				mock.ExpectSMembers("names:found").SetVal([]string{"1", "2"})
				mock.ExpectHGetAll("post:1").SetVal(map[string]string{
					"name":       "found",
					"author":     "vt",
					"created_at": "1",
				})
				mock.ExpectHGetAll("post:2").SetVal(map[string]string{
					"name":       "found",
					"author":     "vt",
					"created_at": "2",
				})
			},
		},
		/*{
			expected: []*Post{},
			err:      false,
			filters:  &SearchFilter{},
			mock: func() {

			},
		},*/
	}

	for _, test := range tests {
		test.mock()

		id, err := ps.FindMany(context.Background(), test.filters)
		if test.err {
			assert.Error(t, err, test.name)
		} else {
			if assert.NoError(t, err) {
				assert.Equal(t, test.expected, id, test.name)
			}
		}

		mock.ClearExpect()
	}
}
