# ChatGPT Conversation Tracker

A lightweight web app + Go backend that lets you import, browse, and manage your ChatGPT export locally. The project ships with a simple CRUD table for session metadata, one-click access to the original chat on chatgpt.com, and a local transcript viewer so you can keep a permanent copy even after deleting the cloud history.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Getting started](#getting-started)
- [File layout](#file-layout)
- [Common tasks](#common-tasks)
- [Notes](#notes)

## Prerequisites

- Go 1.25 or newer (bundled toolchain works fine on macOS/Linux).
- ChatGPT data export (`conversations.json`) from **Settings & Beta → Data Controls → Export data**. Place it in the project root or pass a custom path during import.

## Getting started

1. Install dependencies and build the binaries (Go modules handle this on first run):
   ```bash
   go build ./...
   ```

2. Import your exported conversations. This will parse the JSON export and populate `data/conversations_store.json` with metadata and full transcripts. The import command is idempotent—you can re-run it after future exports to refresh your archive.
   ```bash
   go run ./cmd/importer -file conversations.json
   ```
   - Use `-data <path>` if you want the local store somewhere else (default is `data/conversations_store.json`).

3. Start the web server (serves both the API and static files). By default it listens on `:8080` and serves the `index.html` page from the project root; override with `-addr` and `-static` if needed.
   ```bash
   go run ./cmd/server -addr :8080 -static .
   ```

4. Open `http://localhost:8080/` in your browser.
   - The landing page lists every imported conversation with metadata, actions to view/rename/delete, and links back to the original chat on chatgpt.com.
   - Use the `View` button to open the built-in transcript page (`conversation.html`), which reads from your local archive even if the cloud copy has been removed.

## File layout

```
.
├── cmd/
│   ├── importer/          # CLI that loads ChatGPT exports into the local store
│   └── server/            # HTTP server exposing the API and static assets
├── internal/
│   ├── api/               # REST handlers (list/create/update/delete/fetch)
│   ├── importer/          # Export parser that normalises JSON → local model
│   ├── models/            # Shared data structures for conversations/messages
│   └── storage/           # JSON-backed persistence with basic CRUD helpers
├── data/
│   └── conversations_store.json # Generated archive (created after import)
├── index.html             # Main UI (CRUD table + add form)
├── conversation.html      # Transcript viewer page
├── script.js              # Front-end logic for the CRUD dashboard
├── conversation.js        # Front-end logic for the transcript viewer
├── styles.css             # Shared styling for both pages
├── conversations.json     # (Optional) ChatGPT export file for importer input
└── README.md
```

## Common tasks

- **Re-import after a new ChatGPT export:**
  ```bash
  go run ./cmd/importer -file /path/to/new/conversations.json
  ```
  Existing records are updated in place; new conversations are appended.

- **Change storage location:**
  ```bash
  go run ./cmd/importer -file conversations.json -data /custom/path/store.json
  go run ./cmd/server -addr :8080 -data /custom/path/store.json -static .
  ```

- **Run with a different static directory:** useful if you host the UI elsewhere but still want the API.
  ```bash
  go run ./cmd/server -static ./public
  ```

## Notes

- The importer pulls the first user or assistant message to build the one-line summary shown in the list view.
- Only user/assistant text turns are stored in the transcript; system/tool messages are skipped for readability.
- The UI is zero-JS-build (plain HTML/CSS/ES modules). Serve it from the Go binary or any other static file host—just point the API calls to the server URL.
- No external dependencies or network calls are required after you have the export; everything runs locally.

Feel free to extend the API with search, tagging, or export routines to fit your workflow.
