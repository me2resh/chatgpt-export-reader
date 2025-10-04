package importer

import (
	"encoding/json"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"zatGPT/internal/models"
)

// LoadAndConvert reads an export file and returns Conversation models ready for persistence.
func LoadAndConvert(path string) ([]models.Conversation, error) {
	payload, err := readExport(path)
	if err != nil {
		return nil, err
	}

	conversations := make([]models.Conversation, 0, len(payload))
	for _, raw := range payload {
		if item := convertConversation(raw); item != nil {
			conversations = append(conversations, *item)
		}
	}

	return conversations, nil
}

type exportConversation struct {
	ID             string                `json:"id"`
	ConversationID string                `json:"conversation_id"`
	Title          string                `json:"title"`
	CreateTime     *float64              `json:"create_time"`
	UpdateTime     *float64              `json:"update_time"`
	CurrentNode    string                `json:"current_node"`
	Mapping        map[string]exportNode `json:"mapping"`
}

type exportNode struct {
	ID       string         `json:"id"`
	Parent   string         `json:"parent"`
	Children []string       `json:"children"`
	Message  *exportMessage `json:"message"`
}

type exportMessage struct {
	ID         string        `json:"id"`
	Author     exportAuthor  `json:"author"`
	CreateTime *float64      `json:"create_time"`
	UpdateTime *float64      `json:"update_time"`
	Content    exportContent `json:"content"`
}

type exportAuthor struct {
	Role string `json:"role"`
}

type exportContent struct {
	ContentType string            `json:"content_type"`
	Parts       []json.RawMessage `json:"parts"`
}

func readExport(path string) ([]exportConversation, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var payload []exportConversation
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func convertConversation(raw exportConversation) *models.Conversation {
	if len(raw.Mapping) == 0 {
		return nil
	}

	timeline := traversalPath(raw)
	if len(timeline) == 0 {
		return nil
	}

	var (
		earliest       time.Time
		latest         time.Time
		hasEarliest    bool
		hasLatest      bool
		firstUser      string
		firstAssistant string
		messages       []models.Message
	)

	for _, node := range timeline {
		if node.Message == nil {
			continue
		}

		if ts, ok := toTime(node.Message.CreateTime); ok {
			if !hasEarliest || ts.Before(earliest) {
				earliest = ts
				hasEarliest = true
			}
			if !hasLatest || ts.After(latest) {
				latest = ts
				hasLatest = true
			}
		}

		text := extractText(node.Message.Content)
		if text == "" {
			continue
		}

		role := strings.ToLower(node.Message.Author.Role)
		switch role {
		case "user":
			if firstUser == "" {
				firstUser = text
			}
			messages = append(messages, models.Message{
				ID:        node.ID,
				Author:    role,
				Content:   text,
				CreatedAt: timestampOrZero(node.Message.CreateTime),
			})
		case "assistant":
			if firstAssistant == "" {
				firstAssistant = text
			}
			messages = append(messages, models.Message{
				ID:        node.ID,
				Author:    role,
				Content:   text,
				CreatedAt: timestampOrZero(node.Message.CreateTime),
			})
		}
	}

	if !hasEarliest {
		if ts, ok := toTime(raw.CreateTime); ok {
			earliest = ts
			hasEarliest = true
		}
	}

	if !hasLatest {
		if ts, ok := toTime(raw.UpdateTime); ok {
			latest = ts
			hasLatest = true
		}
	}

	summary := firstNonEmpty(firstUser, firstAssistant)
	summary = truncate(summary, 240)
	if summary == "" {
		summary = "No summary available"
	}

	title := strings.TrimSpace(raw.Title)
	if title == "" {
		title = truncate(summary, 80)
		if title == "" {
			title = "Untitled conversation"
		}
	}

	var dateStarted, dateEnded string
	var createdAt, updatedAt time.Time

	if hasEarliest {
		dateStarted = earliest.UTC().Format("2006-01-02")
		createdAt = earliest.UTC()
	} else {
		createdAt = time.Now().UTC()
	}

	if hasLatest {
		dateEnded = latest.UTC().Format("2006-01-02")
		updatedAt = latest.UTC()
	} else {
		updatedAt = createdAt
	}

	id := strings.TrimSpace(raw.ConversationID)
	if id == "" {
		id = strings.TrimSpace(raw.ID)
	}
	if id == "" {
		id = newDeterministicID(title, createdAt)
	}

	return &models.Conversation{
		ID:          id,
		Title:       title,
		Summary:     summary,
		DateStarted: dateStarted,
		DateEnded:   dateEnded,
		SourceID:    id,
		Messages:    messages,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func traversalPath(raw exportConversation) []exportNode {
	if raw.CurrentNode == "" {
		return timelineByTimestamps(raw)
	}

	path := make([]exportNode, 0, len(raw.Mapping))
	seen := make(map[string]bool)
	nodeID := raw.CurrentNode

	for nodeID != "" {
		node, ok := raw.Mapping[nodeID]
		if !ok {
			break
		}
		path = append(path, node)
		if seen[nodeID] {
			break
		}
		seen[nodeID] = true
		nodeID = node.Parent
	}

	// reverse to chronological order from root to leaf
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	if len(path) == 0 {
		return timelineByTimestamps(raw)
	}

	return path
}

func timelineByTimestamps(raw exportConversation) []exportNode {
	nodes := make([]exportNode, 0, len(raw.Mapping))
	for _, node := range raw.Mapping {
		nodes = append(nodes, node)
	}

	sort.Slice(nodes, func(i, j int) bool {
		ti, okI := toTime(nodes[i].Message.GetCreateTime())
		tj, okJ := toTime(nodes[j].Message.GetCreateTime())
		if okI && okJ {
			if ti.Equal(tj) {
				return nodes[i].ID < nodes[j].ID
			}
			return ti.Before(tj)
		}
		if okI {
			return true
		}
		if okJ {
			return false
		}
		return nodes[i].ID < nodes[j].ID
	})

	return nodes
}

func extractText(content exportContent) string {
	switch content.ContentType {
	case "text":
		return collectStringParts(content.Parts)
	case "multimodal_text":
		return collectStringParts(content.Parts)
	default:
		return ""
	}
}

func collectStringParts(parts []json.RawMessage) string {
	var builder strings.Builder
	for _, part := range parts {
		var text string
		if err := json.Unmarshal(part, &text); err == nil {
			cleaned := strings.TrimSpace(text)
			if cleaned == "" {
				continue
			}
			if builder.Len() > 0 {
				builder.WriteString("\n\n")
			}
			builder.WriteString(cleaned)
		}
	}
	return strings.TrimSpace(builder.String())
}

func truncate(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	trimmed := strings.TrimSpace(text[:limit])
	if !strings.HasSuffix(trimmed, "...") {
		trimmed += "..."
	}
	return trimmed
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func toTime(value *float64) (time.Time, bool) {
	if value == nil {
		return time.Time{}, false
	}
	seconds, frac := math.Modf(*value)
	t := time.Unix(int64(seconds), int64(frac*1e9)).UTC()
	if t.IsZero() {
		return time.Time{}, false
	}
	return t, true
}

func timestampOrZero(value *float64) time.Time {
	if t, ok := toTime(value); ok {
		return t
	}
	return time.Time{}
}

func newDeterministicID(seed string, t time.Time) string {
	base := strings.ReplaceAll(strings.ToLower(seed), " ", "-")
	if base == "" {
		base = "conversation"
	}
	return base + "-" + t.Format("20060102150405")
}

func (m *exportMessage) GetCreateTime() *float64 {
	if m == nil {
		return nil
	}
	return m.CreateTime
}
