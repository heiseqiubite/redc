package mod

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// MemoryItem represents a single agent memory entry
type MemoryItem struct {
	ID        int    `json:"id"`
	Project   string `json:"project"`
	Category  string `json:"category"` // "lesson" | "preference" | "note"
	Content   string `json:"content"`
	Source    string `json:"source"` // "auto" or conversationId
	CreatedAt string `json:"createdAt"`
}

const memoryMaxItems = 50

// MemoryStore manages agent memory persistence
type MemoryStore struct {
	mu sync.Mutex
	db *sql.DB
}

// NewMemoryStore creates and initializes a memory store
func NewMemoryStore() (*MemoryStore, error) {
	if err := ensureRedcPath(); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(RedcPath, "memory.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open memory db: %v", err)
	}

	createSQL := `
	CREATE TABLE IF NOT EXISTS agent_memory (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project TEXT NOT NULL,
		category TEXT NOT NULL,
		content TEXT NOT NULL,
		source TEXT DEFAULT 'auto',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_memory_project ON agent_memory(project);
	`
	if _, err := db.Exec(createSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create memory table: %v", err)
	}

	return &MemoryStore{db: db}, nil
}

// Close closes the database connection
func (m *MemoryStore) Close() {
	if m.db != nil {
		m.db.Close()
	}
}

// AddMemory adds a new memory entry and enforces the max items limit
func (m *MemoryStore) AddMemory(project, category, content, source string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate content
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM agent_memory WHERE project = ? AND content = ?", project, content).Scan(&count)
	if err == nil && count > 0 {
		return nil // skip duplicate
	}

	_, err = m.db.Exec(
		"INSERT INTO agent_memory (project, category, content, source, created_at) VALUES (?, ?, ?, ?, ?)",
		project, category, content, source, time.Now().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return err
	}

	// Enforce max items: delete oldest entries beyond limit
	m.db.Exec(`
		DELETE FROM agent_memory WHERE project = ? AND id NOT IN (
			SELECT id FROM agent_memory WHERE project = ? ORDER BY created_at DESC LIMIT ?
		)
	`, project, project, memoryMaxItems)

	return nil
}

// ListMemories returns all memories for a given project
func (m *MemoryStore) ListMemories(project string) ([]MemoryItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rows, err := m.db.Query(
		"SELECT id, project, category, content, source, created_at FROM agent_memory WHERE project = ? ORDER BY created_at DESC",
		project,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []MemoryItem
	for rows.Next() {
		var item MemoryItem
		if err := rows.Scan(&item.ID, &item.Project, &item.Category, &item.Content, &item.Source, &item.CreatedAt); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// DeleteMemory deletes a specific memory by ID
func (m *MemoryStore) DeleteMemory(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, err := m.db.Exec("DELETE FROM agent_memory WHERE id = ?", id)
	return err
}

// ClearMemories deletes all memories for a project
func (m *MemoryStore) ClearMemories(project string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, err := m.db.Exec("DELETE FROM agent_memory WHERE project = ?", project)
	return err
}

// GetMemoryContext builds a formatted string of memories for prompt injection
func (m *MemoryStore) GetMemoryContext(project string) string {
	items, err := m.ListMemories(project)
	if err != nil || len(items) == 0 {
		return ""
	}

	var lessons, preferences []string
	for _, item := range items {
		switch item.Category {
		case "lesson":
			lessons = append(lessons, item.Content)
		case "preference":
			preferences = append(preferences, item.Content)
		default:
			lessons = append(lessons, item.Content)
		}
	}

	result := ""
	if len(preferences) > 0 {
		result += "### 用户偏好\n"
		for _, p := range preferences {
			result += fmt.Sprintf("- %s\n", p)
		}
	}
	if len(lessons) > 0 {
		result += "### 历史经验教训\n"
		for i, l := range lessons {
			result += fmt.Sprintf("%d. %s\n", i+1, l)
		}
	}
	return result
}
