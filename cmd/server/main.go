package main

import (
    "flag"
    "log"
    "net/http"
    "os"
    "time"

    "zatGPT/internal/api"
    "zatGPT/internal/storage"
)

func main() {
    addr := flag.String("addr", ":8080", "HTTP listen address")
    dataPath := flag.String("data", "data/conversations_store.json", "path to persistence file")
    staticDir := flag.String("static", ".", "directory for serving static assets")
    flag.Parse()

    store, err := storage.New(*dataPath)
    if err != nil {
        log.Fatalf("failed to initialize storage: %v", err)
    }

    mux := http.NewServeMux()

    apiServer := api.New(store)
    apiServer.Register(mux)

    fileServer := http.FileServer(http.Dir(*staticDir))
    mux.Handle("/", fileServer)

    server := &http.Server{
        Addr:         *addr,
        Handler:      withCORS(mux),
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    log.Printf("listening on %s", *addr)
    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Printf("server error: %v", err)
        os.Exit(1)
    }
}

func withCORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,PATCH,OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent)
            return
        }

        next.ServeHTTP(w, r)
    })
}
