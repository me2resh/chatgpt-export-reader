package main

import (
    "flag"
    "fmt"
    "log"

    "zatGPT/internal/importer"
    "zatGPT/internal/storage"
)

func main() {
    filePath := flag.String("file", "conversations.json", "path to ChatGPT export JSON")
    dataPath := flag.String("data", "data/conversations_store.json", "destination persistence file")
    flag.Parse()

    items, err := importer.LoadAndConvert(*filePath)
    if err != nil {
        log.Fatalf("failed to parse export: %v", err)
    }

    store, err := storage.New(*dataPath)
    if err != nil {
        log.Fatalf("failed to open store: %v", err)
    }

    var created, updated int
    for _, item := range items {
        if _, err := store.Get(item.ID); err == nil {
            updated++
        } else {
            created++
        }
        if err := store.Upsert(item); err != nil {
            log.Fatalf("failed to persist conversation %s: %v", item.ID, err)
        }
    }

    fmt.Printf("Imported %d conversations (%d new, %d updated)\n", len(items), created, updated)
}
