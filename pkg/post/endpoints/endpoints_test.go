package endpoints

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
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

	switch filters.Order {
	case post.Ascending:
		sort.Slice(posts, func(i, j int) bool {
			return posts[i].CreatedAt.Before(posts[j].CreatedAt)
		})
	case post.Descending:
		sort.Slice(posts, func(i, j int) bool {
			return posts[i].CreatedAt.After(posts[j].CreatedAt)
		})
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

		assert.Equal(t, test.expectedBody, body, test.name)
		assert.Equal(t, test.expectedStatus, rec.Code, test.name)
	}
}

func TestAddEndpoint(t *testing.T) {
	svc := serviceMock{}
	ep := makeAddEndpoint(svc)
	r := mux.NewRouter()
	r.HandleFunc("/api/posts", ep).Methods("POST")

	tests := []struct {
		name           string
		body           string
		expectedBody   string
		expectedStatus int
	}{
		{
			name:           "Success.",
			body:           `{"name":"test","author":"vt","created_at":"2021-02-28T20:15:24.596Z"}`,
			expectedBody:   `{"status":200,"message":"Successfully created a new post.","id":1}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Badly formatted JSON. Trailing coma.",
			body:           `{"name":"test","author":"vt","created_at":"2021-02-28T20:15:24.596Z",}`,
			expectedBody:   `{"status":400,"message":"Request body contains badly-formatted JSON."}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Bad request. Unknown fields.",
			body:           `{"name":"test","unknown_field":true}`,
			expectedBody:   `{"status":400,"message":"Request body contains unknown field \"unknown_field\""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty body.",
			body:           ``,
			expectedBody:   `{"status":400,"message":"Request body is empty"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty name",
			body:           `{"name":"","author":"vt","created_at":"2021-02-28T20:15:24.596Z"}`,
			expectedBody:   `{"status":400,"message":"name field cannot be empty."}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty author",
			body:           `{"name":"535","author":"","created_at":"2021-02-28T20:15:24.596Z"}`,
			expectedBody:   `{"status":400,"message":"author field cannot be empty."}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/posts", strings.NewReader(test.body))
		req.Header.Set("Content-Type", "application/json")

		r.ServeHTTP(rec, req)
		body := rec.Body.String()

		assert.Equal(t, test.expectedBody, body, test.name)
		assert.Equal(t, test.expectedStatus, rec.Code, test.name)
	}
}
