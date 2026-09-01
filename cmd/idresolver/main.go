package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Exergism-Commons/id.exergism-commons.github.io/internal/resolver"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8080", "HTTP listen address")
	root := flag.String("root", ".", "repository/static root")
	registry := flag.String("registry", "resolver/registry.json", "resolver registry file")
	flag.Parse()

	handler, err := resolver.Load(*root, *registry)
	if err != nil {
		log.Fatalf("load resolver: %v", err)
	}

	server := &http.Server{
		Addr:              *listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          log.New(os.Stderr, "idresolver: ", log.LstdFlags),
	}

	log.Printf("id.exergism.org resolver listening on %s", *listen)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}
