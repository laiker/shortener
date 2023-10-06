package main

import (
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_decodeHandler(t *testing.T) {
	type want struct {
		query    string
		method   string
		code     int
		response string
	}

	tests := []struct {
		name   string
		want   want
		errMsg string
	}{
		{
			"Success Test",
			want{
				"/aHR0cHM6Ly9hc2QucnU=",
				http.MethodGet,
				http.StatusTemporaryRedirect,
				"https://asd.ru",
			},
			"",
		},
		{
			"Error Method Get",
			want{
				"/aHR0cHM6Ly9hc2Qucn=",
				http.MethodPost,
				http.StatusMethodNotAllowed,
				"",
			},
			"Expected method not allowed",
		},
		{
			"Wrong ID",
			want{
				"/",
				http.MethodGet,
				http.StatusNotFound,
				"404 page not found\n",
			},
			"Expected not found",
		},
		{
			"Wrong Url",
			want{
				"/aHR0cHM6Ly9hc2Qucn=",
				http.MethodGet,
				http.StatusBadRequest,
				"Error: wrong decode\n",
			},
			"Expected bad request",
		},
	}

	router := chi.NewRouter()
	router.HandleFunc("/{id}", decodeHandler)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.want.method, tt.want.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, request)

			if tt.want.code == http.StatusTemporaryRedirect {
				assert.Equal(t, tt.want.response, w.Header().Get("Location"))
			} else {
				assert.Equal(t, tt.want.response, w.Body.String())
			}

			assert.Equal(t, tt.want.code, w.Code)
		})
	}
}

func Test_encodeHandler(t *testing.T) {
	type want struct {
		query    string
		method   string
		code     int
		body     string
		response string
	}

	tests := []struct {
		name string
		want want
	}{
		{
			"Success Test",
			want{
				"/",
				http.MethodPost,
				http.StatusCreated,
				"https://asd.ru",
				"http://example.com/aHR0cHM6Ly9hc2QucnU=",
			},
		},
		{
			"Wrong Request",
			want{
				"/",
				http.MethodGet,
				http.StatusMethodNotAllowed,
				"",
				"",
			},
		},
		{
			"Wrong Url",
			want{
				"/",
				http.MethodPost,
				http.StatusBadRequest,
				"xcvcxv",
				"Invalid Url\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			request := httptest.NewRequest(tt.want.method, tt.want.query, strings.NewReader(tt.want.body))
			w := httptest.NewRecorder()
			encodeHandler(w, request)
			result := w.Result()

			defer result.Body.Close()
			respBody, _ := io.ReadAll(result.Body)

			assert.Equal(t, tt.want.response, string(respBody))
			assert.Equal(t, tt.want.code, result.StatusCode)
		})
	}
}
