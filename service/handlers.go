package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
)

var pageSize = 1000

func (s *Service) handleGetFavourites() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.ParseUint(vars["id"], 10, 64)
		if err != nil {
			s.log.Printf("Error parsing user ID: %v", err)
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		page, limit := -1, -1

		pageStr := r.URL.Query().Get("page")
		if pageStr != "" {
			page, err = strconv.Atoi(pageStr)
			if err != nil {
				s.log.Printf("Error parsing page: %v", err)
				http.Error(w, "Invalid page number", http.StatusBadRequest)
				return
			}
		}

		limitStr := r.URL.Query().Get("limit")
		if limitStr != "" {
			limit, err = strconv.Atoi(limitStr)
			if err != nil {
				s.log.Printf("Error parsing limit: %v", err)
				http.Error(w, "Invalid limit", http.StatusBadRequest)
				return
			}
		}

		typeStr := r.URL.Query().Get("type")
		assets, err := s.storage.userFavourites(uint(id), assetType(typeStr))
		if err != nil {
			s.log.Printf("Error getting favourites: %v", err)
			http.Error(w, "Error geting favourites assets from storage", http.StatusInternalServerError)
			return
		}
		// XXX: For efficiency, we could also apply limit and pagination
		// inside the storage layer.
		s.log.Printf("Got %d favourite assets for user %d", len(assets), id)
		if page >= 0 {
			start := min(page*pageSize, len(assets))
			end := min((page+1)*pageSize, len(assets))
			assets = assets[start:end]
			s.log.Printf("Applied pagination with start %d and end %d. Favourites len now is %d", start, end, len(assets))
		}
		if limit >= 0 {
			assets = assets[:min(limit, len(assets))]
			s.log.Printf("Limited favourites to %d", len(assets))
		}
		w.Header().Set("Content-Type", "application/json")
		// Stream the response in chunks rather than encoding the whole
		// thing into memory. Under the hood, w buffers the response
		// bytes to the wire. The same techinique is used in GetAssets
		// and GetUsers handlers.
		err = json.NewEncoder(w).Encode(assets)
		if err != nil {
			// Before the error occurs, we may have already sent
			// some payload to the client. If that's the case,
			// response headers are already set (i.e 200 OK).
			// Clients can detect failure by not receiving a
			// zero-length chunk in chunked transfer encoding.
			// https://stackoverflow.com/questions/17203379/response-sent-in-chunked-transfer-encoding-and-indicating-errors-happening-after
			s.log.Printf("Error sending asset: %v", err)
			http.Error(w, "Error marshalling asset", http.StatusInternalServerError)
			return
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *Service) handleAddFavourite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			s.log.Printf("Error parsing user ID: %v", err)
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}
		assetID, err := strconv.Atoi(vars["assetID"])
		if err != nil {
			s.log.Printf("Error parsing asset ID: %v", err)
			http.Error(w, "Invalid asset ID", http.StatusBadRequest)
			return
		}
		err = s.storage.addFavourites(uint(id), uint(assetID))
		if err != nil {
			s.log.Printf("Error adding favourite: %v", err)
			if errors.Is(err, ErrExist) {
				http.Error(w, "Asset already favourited", http.StatusConflict)
				return
			}
			http.Error(w, "Error adding favourite", http.StatusInternalServerError)
			return
		}
		s.metrics.Add("assets_is_favourite", 1)
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Service) handleDeleteFavourite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			s.log.Printf("Error parsing user ID: %v", err)
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}
		assetID, err := strconv.Atoi(vars["assetID"])
		if err != nil {
			s.log.Printf("Error parsing asset ID: %v", err)
			http.Error(w, "Invalid asset ID", http.StatusBadRequest)
			return
		}
		s.storage.deleteFavourite(uint(id), uint(assetID))
		if err != nil {
			s.log.Printf("Error deleting favourite: %v", err)
			http.Error(w, "Error deleting favourite", http.StatusInternalServerError)
			return
		}
		s.metrics.Add("assets_is_favourite", -1)
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Service) handleGetAsset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			s.log.Printf("Error parsing asset ID: %v", err)
			http.Error(w, "Invalid asset ID", http.StatusBadRequest)
			return
		}

		asset, err := s.storage.assetWithRLock(uint(id))
		if err != nil {
			s.log.Printf("Error getting asset: %v", err)
			http.Error(w, "Asset not found", http.StatusNotFound)
			return
		}
		assetBytes, err := json.Marshal(asset)
		if err != nil {
			s.log.Printf("Error marshalling asset: %v", err)
			http.Error(w, "Error marshalling asset", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(assetBytes)
	}
}

func (s *Service) handleGetAssets() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		typeStr := r.URL.Query().Get("type")
		assets := s.storage.getAssets(assetType(typeStr))
		err := json.NewEncoder(w).Encode(assets)
		if err != nil {
			// Before the error occurs, we may have already sent
			// some payload to the client. If that's the case,
			// response headers are already set (i.e 200 OK).
			// Clients can detect failure by not receiving a
			// zero-length chunk in chunked transfer encoding.
			// https://stackoverflow.com/questions/17203379/response-sent-in-chunked-transfer-encoding-and-indicating-errors-happening-after
			s.log.Printf("Error sending assets: %v", err)
			http.Error(w, "Error marshalling assets", http.StatusInternalServerError)
			return
		}

	}
}

func (s *Service) handleCreateAsset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var a asset
		err := json.NewDecoder(r.Body).Decode(&a)
		if err != nil {
			s.log.Printf("Error decoding asset: %v", err)
			http.Error(w, "Error decoding asset", http.StatusBadRequest)
			return
		}
		err = a.validate()
		if err != nil {
			msg := fmt.Sprintf("Error validating asset: %v", err)
			s.log.Print(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		s.storage.addAsset(&a)
		s.metrics.Add("assets", 1)
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Service) handleUpdateAsset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			s.log.Printf("Error parsing asset ID: %v", err)
			http.Error(w, "Invalid asset ID", http.StatusBadRequest)
			return
		}

		var a asset
		err = json.NewDecoder(r.Body).Decode(&a)
		if err != nil {
			s.log.Printf("Error decoding asset: %v", err)
			http.Error(w, "Error decoding asset", http.StatusBadRequest)
			return
		}
		a.ID = uint(id)

		err = s.storage.updateAsset(&a)
		if err != nil {
			s.log.Printf("Error updating asset: %v", err)
			http.Error(w, "Asset ID not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Service) handleDeleteAsset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			s.log.Printf("Error parsing asset ID: %v", err)
			http.Error(w, "Invalid asset ID", http.StatusBadRequest)
			return
		}
		s.storage.deleteAsset(uint(id))
		s.metrics.Add("assets", -1)
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Service) handleGetUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			s.log.Printf("Error parsing user ID: %v", err)
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		user, err := s.storage.userWithRLock(uint(id))
		if err != nil {
			s.log.Printf("Error getting user: %v", err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		userBytes, err := json.Marshal(user)
		if err != nil {
			s.log.Printf("Error marshalling user: %v", err)
			http.Error(w, "Error marshalling user", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(userBytes)
	}
}

func (s *Service) handleGetUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		users := s.storage.getUsers()
		err := json.NewEncoder(w).Encode(users)
		if err != nil {
			// Before the error occurs, we may have already sent
			// some payload to the client. If that's the case,
			// response headers are already set (i.e 200 OK).
			// Clients can detect failure by not receiving a
			// zero-length chunk in chunked transfer encoding.
			// https://stackoverflow.com/questions/17203379/response-sent-in-chunked-transfer-encoding-and-indicating-errors-happening-after
			s.log.Printf("Error sending users: %v", err)
			http.Error(w, "Error marshalling users", http.StatusInternalServerError)
			return
		}
	}
}

func (s *Service) handleCreateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u user
		err := json.NewDecoder(r.Body).Decode(&u)
		if err != nil {
			s.log.Printf("Error decoding user: %v", err)
			http.Error(w, "Error decoding user", http.StatusBadRequest)
			return
		}
		err = u.validate()
		if err != nil {
			msg := fmt.Sprintf("Error validating user: %v", err)
			s.log.Print(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		s.storage.addUser(&u)
		s.metrics.Add("users", 1)
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Service) handleUpdateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			s.log.Printf("Error parsing user ID: %v", err)
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		var u user
		err = json.NewDecoder(r.Body).Decode(&u)
		if err != nil {
			s.log.Printf("Error decoding user: %v", err)
			http.Error(w, "Error decoding user", http.StatusBadRequest)
			return
		}
		u.ID = uint(id)

		err = s.storage.updateUser(&u)
		if err != nil {
			s.log.Printf("Error updating user: %v", err)
			http.Error(w, "User ID not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Service) handleDeleteUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			s.log.Printf("Error parsing user ID: %v", err)
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}
		s.storage.deleteUser(uint(id))
		s.metrics.Add("users", 1)
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Service) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

// a struct that will be encoded to a JWT.
type jwtClaims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

var users = map[string]string{
	"username1": "password1",
	"gwi":       "gwi",
}

var secretKey = []byte("tYrkJq2j28yM8kdX_H_rUith1Qx18LaBFqhJJ6m0wuTJrfWdX8CNQOP7xR7pr5j9eOLk2SEq113y80AmJ6g8d2tW0X6xP6JEL7jlkskZZQmNM7cx90AhZlv5nDNsbLie")

func (s *Service) handleLogin() http.HandlerFunc {
	type credentials struct {
		Password string `json:"password"`
		Username string `json:"username"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var creds credentials
		err := json.NewDecoder(r.Body).Decode(&creds)
		if err != nil {
			s.log.Printf("Error decoding credentials: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		pswd, ok := users[creds.Username]
		if !ok || pswd != creds.Password {
			s.log.Printf("Invalid credentials: %v", creds)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		expireTime := time.Now().Add(24 * time.Hour)
		claims := &jwtClaims{
			Username: creds.Username,
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: expireTime.Unix(),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		// Generated from https://mkjwk.org/
		tokenString, err := token.SignedString(secretKey)
		if err != nil {
			s.log.Printf("Error signing token: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:    "token",
			Value:   tokenString,
			Expires: expireTime,
		})
		w.WriteHeader(http.StatusOK)
	}
}
