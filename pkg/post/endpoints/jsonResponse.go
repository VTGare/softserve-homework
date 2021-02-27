package endpoints

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type jsonResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type newPostResp struct {
	jsonResp
	ID int64 `json:"id"`
}

type countResp struct {
	Count   int            `json:"total_count"`
	Authors []*authorCount `json:"authors"`
}

type authorCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

//Error satisfies in-built Error interface
func (r *jsonResp) Error() string {
	return r.Message
}

//responseWriter is a http.ResponseWriter wrapper that adds JSON decoding and encoding methods.
type responseWriter struct {
	http.ResponseWriter
}

//JSON encodes src, writes it to the ResponseWriter and changes Content-Type header to "application/json"
//
//Status is 200 by default, you can optionally overwrite it by passing a second argument.
func (w *responseWriter) JSON(src interface{}, status ...int) {
	msg, err := json.Marshal(src)
	if err != nil {
		w.WriteHeader(500)
		w.Header().Set("Content-Type", "application/json")

		msg, _ := json.Marshal(jsonResp{500, err.Error()})
		w.Write(msg)

		return
	}

	if len(status) != 0 {
		w.WriteHeader(status[0])
	} else {
		w.WriteHeader(200)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(msg)
}

//decodeJSONBody decodes JSON from http.Request.Body to dst. This function errors if any of the following is true:
//
//- Content-Type is not application/json,
//
//- Body size is larger than 1MB,
//
//- Body contains badly formatted JSON,
//
//- JSON contains unknown fields
//
//- Body is empty
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	if r.Header.Get("Content-Type") != "application/json" {
		return &jsonResp{http.StatusUnsupportedMediaType, "Content-Type header is not application/json"}
	}

	//Limit JSON body size to 1MB.
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&dst)
	if err != nil {
		var (
			syntaxError    *json.SyntaxError
			unmarshalError *json.UnmarshalTypeError
		)

		switch {
		case errors.As(err, &syntaxError) || errors.Is(err, io.ErrUnexpectedEOF):
			return &jsonResp{http.StatusBadRequest, "Request body contains badly-formatted JSON."}

		case errors.As(err, &unmarshalError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalError.Field, unmarshalError.Offset)
			return &jsonResp{http.StatusBadRequest, msg}

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)

			return &jsonResp{http.StatusBadRequest, msg}

		case errors.Is(err, io.EOF):
			return &jsonResp{http.StatusBadRequest, "Request body is empty"}

		case err.Error() == "http: request body too large":
			return &jsonResp{http.StatusRequestEntityTooLarge, "Request body shouldn't be larger than 1MB."}

		default:
			return err
		}
	}

	return nil
}
