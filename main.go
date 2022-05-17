package main

import (
	"context"
	"flag"
	"gwitha/service"
	"log"
	"os"
	"os/signal"
	"time"
)

var port = flag.String("port", "8080", "port to listen on")
var ratelimitEnable = flag.Bool("ratelimit", false, "enable rate limiting")

// TODO: use the same logger throughout the whole application
func main() {
	flag.Parse()
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	svc, err := service.New(logger, *port, *ratelimitEnable, true)
	if err != nil {
		logger.Fatal(err)
	}
	// Run our server in a goroutine so that it doesn't block.
	logger.Printf("Starting GWI service on port %s", *port)
	go func() {
		if err := svc.ListenAndServe(); err != nil {
			logger.Println(err)
		}
	}()
	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c
	logger.Println("Stopping GWI service ...")
	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	svc.Shutdown(ctx)
	os.Exit(0)
}
