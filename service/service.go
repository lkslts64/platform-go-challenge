package service

import (
	"expvar"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type Service struct {
	*http.Server    // embed so we can call http.Server's methods directly
	log             *log.Logger
	router          *mux.Router
	storage         *storage
	enableRatelimit bool
	metrics         *expvar.Map
}

func New(logger *log.Logger, port string, ratelimit, storagePreloadObjects bool) (*Service, error) {
	r := mux.NewRouter()

	srv := &http.Server{
		Addr: fmt.Sprintf("0.0.0.0:%s", port),
		// Typically, I would take this values from a configuration file.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}

	svc := &Service{
		Server:          srv,
		router:          r,
		storage:         newStorage(),
		log:             logger,
		enableRatelimit: ratelimit,
	}
	svc.registerRoutes()

	metrics := expvar.NewMap("metrics")
	metrics.Set("auth_failures", new(expvar.Int))
	metrics.Set("ratelimited_reqs", new(expvar.Int))
	metrics.Set("assets", new(expvar.Int))
	metrics.Set("users", new(expvar.Int))
	metrics.Set("assets_is_favourite", new(expvar.Int))
	svc.metrics = metrics

	if storagePreloadObjects {
		svc.metrics.Add("assets", int64(storageNumberAssetsPreload))
		svc.metrics.Add("assets_is_favourite", int64(storageNumberAssetsPreload))
		svc.metrics.Add("users", 1)
		err := svc.storage.fillWithObjects()
		if err != nil {
			return nil, err
		}
	}

	return svc, nil
}
