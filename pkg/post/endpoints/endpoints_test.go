package endpoints

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/VTGare/softserve-homework/pkg/post"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type serviceMock struct{}

func (m serviceMock) Create(_ context.Context, _ *post.Post) (int64, error) {
	return 1, nil
}

func (m serviceMock) FindOne(_ context.Context, id int64) (*post.Post, error) {
	posts := map[int64]*post.Post{
		1: {1, "test1", "vt", time.Unix(1, 0)},
		2: {2, "test2", "vt", time.Unix(1, 0)},
		3: {3, "test3", "vt", time.Unix(1, 0)},
	}

	post, ok := posts[id]
	if !ok {
		return nil, errors.New("not found")
	}

	return post, nil
}

func (m serviceMock) FindMany(_ context.Context, filters *post.SearchFilter) ([]*post.Post, error) {
	postsMap := map[int64]*post.Post{
		1: {1, "test", "vt", time.Unix(1, 0)},
		2: {2, "test", "robot", time.Unix(1, 0)},
		3: {3, "test2", "vt", time.Unix(1, 0)},
	}

	posts := make([]*post.Post, 0)
	switch {
	case filters.Author != "" && filters.Name != "":
		for _, post := range postsMap {
			if post.Author == filters.Author && post.Name == filters.Name {
				posts = append(posts, post)
			}
		}
	case filters.Author != "":
		for _, post := range postsMap {
			if post.Author == filters.Author {
				posts = append(posts, post)
			}
		}
	case filters.Name != "":
		for _, post := range postsMap {
			if post.Name == filters.Name {
				posts = append(posts, post)
			}
		}
	default:
		for _, post := range postsMap {
			posts = append(posts, post)
		}
	}

	return posts, nil
}

func (m serviceMock) Remove(_ context.Context, id int64) (bool, error) {
	posts := map[int64]*post.Post{
		1: {1, "test1", "vt", time.Unix(1, 0)},
		2: {2, "test2", "vt", time.Unix(1, 0)},
		3: {3, "test3", "vt", time.Unix(1, 0)},
	}

	if _, ok := posts[id]; !ok {
		return false, errors.New("not found")
	}

	return true, nil
}

func (m serviceMock) Logger() *zap.SugaredLogger {
	return zap.NewExample().Sugar()
}

func (m serviceMock) Count(_ context.Context) (map[string]int, error) {
	return map[string]int{
		"vt":    2,
		"robot": 1,
	}, nil
}

func TestGetEndpoint(t *testing.T) {
	svc := serviceMock{}
	ep := makeGetEndpoint(svc)
	r := mux.NewRouter()
	r.HandleFunc("/api/posts/{id}", ep)

	tests := []struct {
		name           string
		id             string
		expectedBody   string
		expectedStatus int
	}{
		{
			name:           "Post 1. Success.",
			id:             "1",
			expectedBody:   fmt.Sprintf(`{"id":1,"name":"test1","author":"vt","created_at":"%v"}`, time.Unix(1, 0).Format(time.RFC3339)),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Non-integer ID. Fail.",
			id:             "pog",
			expectedBody:   `{"status":400,"message":"unable to parse an ID."}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Post 4. Fail.",
			id:             "4",
			expectedBody:   `{"status":404,"message":"post 4 was not found."}`,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/posts/"+test.id, nil)

		r.ServeHTTP(rec, req)
		body := rec.Body.String()

		assert.Equal(t, test.expectedBody, body)
		assert.Equal(t, test.expectedStatus, rec.Code)
	}
}
