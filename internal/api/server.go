package api

import (
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"

    "zatGPT/internal/models"
    "zatGPT/internal/storage"
)

// Server wraps the HTTP handlers for the conversations API.
type Server struct {
    store *storage.Store
}

// New creates a new Server instance.
func New(store *storage.Store) *Server {
    return &Server{store: store}
}

// Register wires the API routes onto the supplied mux.
func (s *Server) Register(mux *http.ServeMux) {
    mux.HandleFunc("/api/conversations", s.handleConversations)
    mux.HandleFunc("/api/conversations/", s.handleConversationByID)
}

func (s *Server) handleConversations(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        s.listConversations(w, r)
    case http.MethodPost:
        s.createConversation(w, r)
    case http.MethodDelete:
        s.deleteAll(w, r)
    default:
        methodNotAllowed(w, http.MethodGet, http.MethodPost, http.MethodDelete)
    }
}

func (s *Server) handleConversationByID(w http.ResponseWriter, r *http.Request) {
    id := strings.TrimPrefix(r.URL.Path, "/api/conversations/")
    id = strings.Trim(id, "/")
    if id == "" {
        http.NotFound(w, r)
        return
    }

    switch r.Method {
    case http.MethodGet:
        s.getConversation(w, r, id)
    case http.MethodPatch:
        s.patchConversation(w, r, id)
    case http.MethodDelete:
        s.deleteConversation(w, r, id)
    default:
        methodNotAllowed(w, http.MethodGet, http.MethodPatch, http.MethodDelete)
    }
}

func (s *Server) listConversations(w http.ResponseWriter, _ *http.Request) {
    writeJSON(w, http.StatusOK, map[string]any{
        "conversations": s.store.List(),
    })
}

func (s *Server) createConversation(w http.ResponseWriter, r *http.Request) {
    var payload struct {
        Title       string `json:"title"`
        Summary     string `json:"summary"`
        DateStarted string `json:"dateStarted"`
        DateEnded   string `json:"dateEnded"`
        SourceID    string `json:"sourceId"`
    }

    if err := decodeJSON(r.Body, &payload); err != nil {
        writeError(w, http.StatusBadRequest, err)
        return
    }

    payload.Title = strings.TrimSpace(payload.Title)
    payload.Summary = strings.TrimSpace(payload.Summary)
    payload.DateStarted = strings.TrimSpace(payload.DateStarted)
    payload.DateEnded = strings.TrimSpace(payload.DateEnded)
    payload.SourceID = strings.TrimSpace(payload.SourceID)

    if payload.Title == "" || payload.Summary == "" {
        writeErrorString(w, http.StatusBadRequest, "title and summary are required")
        return
    }

    convo := models.Conversation{
        ID:          newID(),
        Title:       payload.Title,
        Summary:     payload.Summary,
        DateStarted: payload.DateStarted,
        DateEnded:   payload.DateEnded,
        SourceID:    payload.SourceID,
    }

    if err := s.store.Upsert(convo); err != nil {
        writeError(w, http.StatusInternalServerError, err)
        return
    }

    writeJSON(w, http.StatusCreated, convo)
}

func (s *Server) getConversation(w http.ResponseWriter, _ *http.Request, id string) {
    convo, err := s.store.Get(id)
    if err != nil {
        if err == storage.ErrNotFound {
            http.NotFound(w, nil)
            return
        }
        writeError(w, http.StatusInternalServerError, err)
        return
    }
    writeJSON(w, http.StatusOK, convo)
}

func (s *Server) patchConversation(w http.ResponseWriter, r *http.Request, id string) {
    var payload struct {
        Title       *string `json:"title"`
        Summary     *string `json:"summary"`
        DateStarted *string `json:"dateStarted"`
        DateEnded   *string `json:"dateEnded"`
    }

    if err := decodeJSON(r.Body, &payload); err != nil && err != io.EOF {
        writeError(w, http.StatusBadRequest, err)
        return
    }

    convo, err := s.store.Get(id)
    if err != nil {
        if err == storage.ErrNotFound {
            http.NotFound(w, r)
            return
        }
        writeError(w, http.StatusInternalServerError, err)
        return
    }

    if payload.Title != nil {
        title := strings.TrimSpace(*payload.Title)
        if title == "" {
            writeErrorString(w, http.StatusBadRequest, "title cannot be empty")
            return
        }
        convo.Title = title
    }

    if payload.Summary != nil {
        summary := strings.TrimSpace(*payload.Summary)
        if summary == "" {
            writeErrorString(w, http.StatusBadRequest, "summary cannot be empty")
            return
        }
        convo.Summary = summary
    }

    if payload.DateStarted != nil {
        convo.DateStarted = strings.TrimSpace(*payload.DateStarted)
    }

    if payload.DateEnded != nil {
        convo.DateEnded = strings.TrimSpace(*payload.DateEnded)
    }

    convo.UpdatedAt = time.Now().UTC()

    if err := s.store.Upsert(convo); err != nil {
        writeError(w, http.StatusInternalServerError, err)
        return
    }

    writeJSON(w, http.StatusOK, convo)
}

func (s *Server) deleteConversation(w http.ResponseWriter, _ *http.Request, id string) {
    if err := s.store.Delete(id); err != nil {
        if err == storage.ErrNotFound {
            http.NotFound(w, nil)
            return
        }
        writeError(w, http.StatusInternalServerError, err)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteAll(w http.ResponseWriter, _ *http.Request) {
    if err := s.store.DeleteAll(); err != nil {
        writeError(w, http.StatusInternalServerError, err)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    enc := json.NewEncoder(w)
    _ = enc.Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
    writeErrorString(w, status, err.Error())
}

func writeErrorString(w http.ResponseWriter, status int, msg string) {
    writeJSON(w, status, map[string]string{"error": msg})
}

func methodNotAllowed(w http.ResponseWriter, allowed ...string) {
    w.Header().Set("Allow", strings.Join(allowed, ", "))
    w.WriteHeader(http.StatusMethodNotAllowed)
}

func decodeJSON(body io.ReadCloser, dest any) error {
    defer body.Close()
    decoder := json.NewDecoder(body)
    decoder.DisallowUnknownFields()
    return decoder.Decode(dest)
}

func newID() string {
    buf := make([]byte, 16)
    if _, err := rand.Read(buf); err != nil {
        return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
    }
    return hex.EncodeToString(buf)
}
