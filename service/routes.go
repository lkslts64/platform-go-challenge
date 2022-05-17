package service

import (
	"expvar"
	"net/http"

	"github.com/gorilla/handlers"
)

func (s *Service) registerRoutes() {
	// Login
	s.router.HandleFunc("/login", s.handleLogin()).Methods(http.MethodPost)

	// Assets
	s.router.Handle("/assets", s.handleCreateAsset()).Methods(http.MethodPost)               // create asset
	s.router.Handle("/assets", s.handleGetAssets()).Methods(http.MethodGet)                  // get assets
	s.router.Handle("/assets/{id:[0-9]+}", s.handleGetAsset()).Methods(http.MethodGet)       // get asset
	s.router.Handle("/assets/{id:[0-9]+}", s.handleDeleteAsset()).Methods(http.MethodDelete) // delete asset
	s.router.Handle("/assets/{id:[0-9]+}", s.handleUpdateAsset()).Methods(http.MethodPut)    // update asset

	// Users
	s.router.Handle("/users", s.handleCreateUser()).Methods(http.MethodPost) // create user
	s.router.Handle("/users", s.handleGetUsers()).Methods(http.MethodGet)
	s.router.Handle("/users/{id:[0-9]+}", s.handleGetUser()).Methods(http.MethodGet)
	s.router.Handle("/users/{id:[0-9]+}", s.handleDeleteUser()).Methods(http.MethodDelete) // delete user
	s.router.Handle("/users/{id:[0-9]+}", s.handleUpdateUser()).Methods(http.MethodPut)    // update user

	// Favourites
	s.router.Handle("/users/{id:[0-9]+}/favourites", s.handleGetFavourites()).Methods(http.MethodGet)
	s.router.Handle("/users/{id:[0-9]+}/favourites/{assetID:[0-9]+}", s.handleDeleteFavourite()).Methods(http.MethodGet) // delete favourites
	s.router.Handle("/users/{id:[0-9]+}/favourites/{assetID:[0-9]+}", s.handleAddFavourite()).Methods(http.MethodPut)    // add to favourites

	// Health
	s.router.Handle("/health", s.handleHealth()).Methods(http.MethodGet)

	// Metrics
	s.router.Handle("/metrics", expvar.Handler())

	loggingMw := func(next http.Handler) http.Handler {
		return handlers.LoggingHandler(s.log.Writer(), next)
	}
	recoveryMw := handlers.RecoveryHandler(handlers.PrintRecoveryStack(true), handlers.RecoveryLogger(s.log))

	s.router.Use(recoveryMw, loggingMw, s.authAndRatelimitMiddleware())
}
