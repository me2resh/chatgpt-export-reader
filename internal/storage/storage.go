package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"zatGPT/internal/models"
)

var ErrNotFound = errors.New("conversation not found")

// Store manages conversation persistence backed by a JSON file.
type Store struct {
	mu            sync.RWMutex
	path          string
	conversations map[string]models.Conversation
}

// New creates or loads a Store located at path.
func New(path string) (*Store, error) {
	s := &Store{
		path:          path,
		conversations: make(map[string]models.Conversation),
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

// List returns all conversations sorted by UpdatedAt descending.
func (s *Store) List() []models.Conversation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]models.Conversation, 0, len(s.conversations))
	for _, item := range s.conversations {
		sanitized := item
		sanitized.Messages = nil
		items = append(items, sanitized)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].Title < items[j].Title
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})

	return items
}

// Get fetches a conversation by id.
func (s *Store) Get(id string) (models.Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.conversations[id]
	if !ok {
		return models.Conversation{}, ErrNotFound
	}
	return item, nil
}

// Upsert inserts or updates a conversation.
func (s *Store) Upsert(conversation models.Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	existing, exists := s.conversations[conversation.ID]
	if exists {
		if conversation.CreatedAt.IsZero() {
			conversation.CreatedAt = existing.CreatedAt
		}
	} else if conversation.CreatedAt.IsZero() {
		conversation.CreatedAt = now
	}

	if conversation.UpdatedAt.IsZero() {
		conversation.UpdatedAt = now
	}

	s.conversations[conversation.ID] = conversation
	return s.saveLocked()
}

// UpdateTitle updates the title of a conversation.
func (s *Store) UpdateTitle(id, title string) (models.Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	convo, ok := s.conversations[id]
	if !ok {
		return models.Conversation{}, ErrNotFound
	}

	convo.Title = title
	convo.UpdatedAt = time.Now().UTC()
	s.conversations[id] = convo

	if err := s.saveLocked(); err != nil {
		return models.Conversation{}, err
	}
	return convo, nil
}

// Delete removes a conversation by id.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.conversations[id]; !ok {
		return ErrNotFound
	}

	delete(s.conversations, id)
	return s.saveLocked()
}

// DeleteAll wipes the store.
func (s *Store) DeleteAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.conversations = make(map[string]models.Conversation)
	return s.saveLocked()
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	file, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	var payload struct {
		Conversations []models.Conversation `json:"conversations"`
	}
	if err := json.NewDecoder(file).Decode(&payload); err != nil {
		return err
	}

	for _, item := range payload.Conversations {
		s.conversations[item.ID] = item
	}

	return nil
}

func (s *Store) saveLocked() error {
	payload := struct {
		Conversations []models.Conversation `json:"conversations"`
	}{
		Conversations: make([]models.Conversation, 0, len(s.conversations)),
	}

	for _, item := range s.conversations {
		payload.Conversations = append(payload.Conversations, item)
	}

	sort.Slice(payload.Conversations, func(i, j int) bool {
		if payload.Conversations[i].UpdatedAt.Equal(payload.Conversations[j].UpdatedAt) {
			return payload.Conversations[i].Title < payload.Conversations[j].Title
		}
		return payload.Conversations[i].UpdatedAt.After(payload.Conversations[j].UpdatedAt)
	})

	tmpPath := s.path + ".tmp"
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(&payload); err != nil {
		file.Close()
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	return os.Rename(tmpPath, s.path)
}
