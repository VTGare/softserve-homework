package endpoints

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/VTGare/softserve-homework/pkg/post"
	"github.com/gorilla/mux"
)

type Set struct {
	GetEndpoint    func(http.ResponseWriter, *http.Request)
	AddEndpoint    func(http.ResponseWriter, *http.Request)
	DeleteEndpoint func(http.ResponseWriter, *http.Request)
	SearchEndpoint func(http.ResponseWriter, *http.Request)
	CountEndpoint  func(http.ResponseWriter, *http.Request)
}

func NewEndpointSet(svc post.Service) *Set {
	return &Set{
		GetEndpoint:    makeGetEndpoint(svc),
		AddEndpoint:    makeAddEndpoint(svc),
		DeleteEndpoint: makeDeleteEndpoint(svc),
		SearchEndpoint: makeSearchEndpoint(svc),
		CountEndpoint:  makeCountEndpoint(svc),
	}
}

func makeGetEndpoint(svc post.Service) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{w}
		vars := mux.Vars(r)

		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			rw.JSON(jsonResp{http.StatusBadRequest, "Error parsing an ID. Provide an integer."}, http.StatusBadRequest)
			return
		}

		post, err := svc.FindOne(r.Context(), id)
		if err != nil {
			switch {
			case strings.Contains(err.Error(), "not found"):
				rw.JSON(jsonResp{http.StatusNotFound, fmt.Sprintf("Post with ID %v is not found", id)}, http.StatusNotFound)
				return
			default:
				rw.JSON(jsonResp{http.StatusInternalServerError, err.Error()}, http.StatusInternalServerError)
				return
			}
		}

		rw.JSON(post)
	}
}

func makeAddEndpoint(svc post.Service) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{w}

		var post post.Post
		err := decodeJSONBody(w, r, &post)
		if err != nil {
			var jr *jsonResp
			if errors.As(err, &jr) {
				rw.JSON(jr, jr.Status)
			} else {
				rw.JSON(&jsonResp{http.StatusInternalServerError, err.Error()}, http.StatusInternalServerError)
			}
			return
		}

		id, err := svc.Create(r.Context(), &post)
		if err != nil {
			rw.JSON(&jsonResp{http.StatusInternalServerError, ""}, http.StatusInternalServerError)
			return
		}

		rw.JSON(newPostResp{
			jsonResp: jsonResp{200, "Succesfully added a post."},
			ID:       id,
		})
	}
}

func makeSearchEndpoint(svc post.Service) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{w}

		//Descending sort by default
		order := post.Descending
		if query := r.URL.Query().Get("order"); query != "" {
			switch query {
			case "asc":
				order = post.Ascending
			case "desc":
				order = post.Descending
			default:
				rw.JSON(jsonResp{http.StatusBadRequest, fmt.Sprintf("Unknown sort option: %v.", query)}, http.StatusBadRequest)
				return
			}
		}

		posts, err := svc.FindMany(r.Context(), &post.SearchFilter{
			Name:   r.URL.Query().Get("name"),
			Author: r.URL.Query().Get("author"),
			Order:  order,
		})
		if err != nil {
			switch {
			case strings.Contains(err.Error(), "no results"):
				rw.JSON(jsonResp{http.StatusNotFound, "No results found with applied filters."}, http.StatusNotFound)
				return
			default:
				rw.JSON(jsonResp{http.StatusInternalServerError, err.Error()}, http.StatusInternalServerError)
				return
			}
		}

		rw.JSON(posts)
	}
}

func makeDeleteEndpoint(svc post.Service) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{w}
		vars := mux.Vars(r)

		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			rw.JSON(jsonResp{http.StatusBadRequest, "Error parsing an ID. Provide an integer."}, http.StatusBadRequest)
			return
		}

		_, err = svc.Remove(r.Context(), id)
		if err != nil {
			switch {
			case strings.Contains(err.Error(), "not found"):
				rw.JSON(jsonResp{http.StatusNotFound, fmt.Sprintf("Post with ID %v is not found", id)}, http.StatusNotFound)
				return
			default:
				rw.JSON(jsonResp{http.StatusInternalServerError, err.Error()}, http.StatusInternalServerError)
				return
			}
		}

		rw.JSON(jsonResp{http.StatusOK, "Successfully removed a post with ID: " + vars["id"]})
	}
}

func makeCountEndpoint(svc post.Service) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{w}

		res, err := svc.Count(r.Context())
		if err != nil {
			rw.JSON(jsonResp{http.StatusInternalServerError, err.Error()}, http.StatusInternalServerError)
		}

		authors := make([]*authorCount, 0, len(res))
		count := 0
		for author, num := range res {
			authors = append(authors, &authorCount{author, num})

			count += num
		}

		rw.JSON(&countResp{
			Count:   count,
			Authors: authors,
		})
	}
}