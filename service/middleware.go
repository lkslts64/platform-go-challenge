package service

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
)

func (s *Service) authAndRatelimitMiddleware() mux.MiddlewareFunc {
	// Init limiters
	limiters := make(map[string]*rate.Limiter, len(users))
	for username := range users {
		// Limit to 1000 request every day.
		limit := rate.Every(time.Hour * 24 / 1000)
		if !s.enableRatelimit {
			limit = rate.Inf
		}
		limiters[username] = rate.NewLimiter(limit, 1)

	}
	// Allow 1000 requests per minute for all unauthenticated users
	limiters["unauthenticated"] = rate.NewLimiter(rate.Limit(time.Minute/1000), 1)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/login" {
				if !limiters["unauthenticated"].Allow() {
					s.metrics.Add("ratelimited_reqs", 1)
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			auth := r.Header.Get("Authorization")
			if auth == "" {
				s.metrics.Add("auth_failures", 1)
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}
			authSlc := strings.Split(auth, " ")
			if len(authSlc) != 2 || authSlc[0] != "Bearer" {
				http.Error(w, "Wrong Authoriazation header format", http.StatusBadRequest)
				return
			}
			var claims jwtClaims
			token := authSlc[1]
			tkn, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("incorrect signing method: %v", t.Header["alg"])
				}
				return secretKey, nil
			})

			if err != nil {
				if errors.Is(err, jwt.ErrSignatureInvalid) {
					s.metrics.Add("auth_failures", 1)
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if !tkn.Valid {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			// ==== User is now authenticated ====

			// Check for ratelimit.
			if !limiters[claims.Username].Allow() {
				s.metrics.Add("ratelimited_reqs", 1)
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

}
