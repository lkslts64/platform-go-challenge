package service

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserFavourites(t *testing.T) {
	storageNumberAssetsPreload = 3000
	svc, err := New(log.Default(), "8080", false, true)
	require.NoError(t, err)

	type test struct {
		name    string
		url     string
		results int
		code    int
	}

	tests := []test{
		{
			name:    "no query params",
			url:     "/users/1/favourites",
			results: storageNumberAssetsPreload,
			code:    http.StatusOK,
		},
		{
			name:    "with limit",
			url:     "/users/1/favourites?limit=10",
			results: 10,
			code:    http.StatusOK,
		},
		{
			name:    "with type",
			url:     "/users/1/favourites?type=chart",
			results: storageNumberAssetsPreload / 3,
			code:    http.StatusOK,
		},
		{
			name:    "with type,limit",
			url:     "/users/1/favourites?type=chart&limit=100",
			results: 100,
			code:    http.StatusOK,
		},
		{
			name:    "with page",
			url:     "/users/1/favourites?page=2",
			results: pageSize,
			code:    http.StatusOK,
		},
		{
			name:    "with page out of bounds",
			url:     "/users/1/favourites?page=99999999",
			results: 0,
			code:    http.StatusOK,
		},
		{
			name:    "with page and limit",
			url:     "/users/1/favourites?page=0&limit=25",
			results: 25,
			code:    http.StatusOK,
		},
		{
			name:    "invalid page",
			url:     "/users/1/favourites?page=invalid&limit=25",
			results: 0,
			code:    http.StatusBadRequest,
		},
	}

	h := svc.handleGetFavourites()

	router := mux.NewRouter()
	router.HandleFunc("/users/{id:[0-9]*}/favourites", h)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := http.NewRequest(http.MethodGet, tt.url, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

			assert.Equal(t, tt.code, w.Code)
			if tt.code != http.StatusOK {
				return
			}
			var assets []*asset
			assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &assets))
			assert.Len(t, assets, tt.results)

		})

	}

}
