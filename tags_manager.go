package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TagsManager handles persistence and suggestion logic for task tags.
// tags.json is stored at %USERPROFILE%\Documents\TaskmemoLogger\tags.json
// so users can hand-edit it if needed.
type TagsManager struct {
	filePath string
	tags     []string // in-memory cache; kept in sync with tags.json
}

// NewTagsManager creates a TagsManager. dataDir is the directory holding tags.json.
func NewTagsManager(dataDir string) (*TagsManager, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	tm := &TagsManager{filePath: filepath.Join(dataDir, "tags.json")}
	// Initialize file if not exist
	if _, err := os.Stat(tm.filePath); os.IsNotExist(err) {
		if err := tm.save([]string{}); err != nil {
			return nil, err
		}
		tm.tags = []string{}
		return tm, nil
	}
	// Load once into memory cache
	tags, err := tm.Load()
	if err != nil {
		return nil, err
	}
	tm.tags = tags
	return tm, nil
}

// Load reads tags from disk.
func (t *TagsManager) Load() ([]string, error) {
	data, err := os.ReadFile(t.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read tags: %w", err)
	}
	var tags []string
	if err := json.Unmarshal(data, &tags); err != nil {
		return nil, fmt.Errorf("failed to parse tags: %w", err)
	}
	return tags, nil
}

// save writes tags atomically (write temp → rename) to prevent data loss.
func (t *TagsManager) save(tags []string) error {
	data, err := json.MarshalIndent(tags, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}
	// Write to temp file in same directory
	tmpPath := t.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write tags temp file: %w", err)
	}
	// Atomic rename
	if err := os.Rename(tmpPath, t.filePath); err != nil {
		return fmt.Errorf("failed to rename tags file: %w", err)
	}
	return nil
}

// Add adds a new tag if it does not already exist (case-insensitive dedup).
// Saves atomically. Updates the in-memory cache only after a successful save.
func (t *TagsManager) Add(tag string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil
	}
	// Case-insensitive duplicate check against cache
	tagLower := strings.ToLower(tag)
	for _, existing := range t.tags {
		if strings.ToLower(existing) == tagLower {
			return nil // already exists
		}
	}
	newTags := append(t.tags, tag)
	// Update cache only after successful atomic save
	if err := t.save(newTags); err != nil {
		return err
	}
	t.tags = newTags
	return nil
}

// GetSuggestions returns tags whose prefix matches (case-insensitive).
// Uses the in-memory cache; no disk I/O.
func (t *TagsManager) GetSuggestions(prefix string) []string {
	if prefix == "" {
		result := make([]string, len(t.tags))
		copy(result, t.tags)
		return result
	}
	prefixLower := strings.ToLower(prefix)
	var result []string
	for _, tag := range t.tags {
		if strings.HasPrefix(strings.ToLower(tag), prefixLower) {
			result = append(result, tag)
		}
	}
	return result
}
