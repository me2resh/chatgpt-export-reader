package models

import "time"

// Conversation holds the metadata we surface in the UI and expose via the API.
type Conversation struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	DateStarted string    `json:"dateStarted"`
	DateEnded   string    `json:"dateEnded"`
	SourceID    string    `json:"sourceId,omitempty"`
	Messages    []Message `json:"messages,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Message struct {
	ID        string    `json:"id"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}
