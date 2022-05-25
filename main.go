package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	allowedUsers := NewAllowedUsers(readEnv("TG_ALLOWED_USERS", ""))
	authService, err := NewAuthService(
		readEnv("TG_BOT_TOKEN", ""),
		14*24*time.Hour,
		allowedUsers)
	if err != nil {
		panic(err)
	}

	backendURL, err := url.Parse(readEnv("BACKEND_URL", ""))
	if err != nil {
		panic(err)
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(backendURL)
	log.Printf("backend url: %v", backendURL)

	mux := http.NewServeMux()
	mux.Handle("/_/login", loginHandler(authService))
	mux.Handle("/_/logout", logoutHandler(authService))
	mux.Handle("/", contentHandler(authService, reverseProxy))

	listenAddr := readEnv("LISTEN_ADDR", "0.0.0.0:8000")
	server := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Printf("listening on %v", server.Addr)

	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("unable to shut down gracefully: %+v", err)
	}
}

func readEnv(key, val string) string {
	s := os.Getenv(key)
	if s == "" {
		s = val
	}

	if s == "" {
		panic(fmt.Sprintf("missing env variable %s", key))
	}

	return s
}
