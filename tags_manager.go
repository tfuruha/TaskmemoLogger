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
	}
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
// Saves atomically.
func (t *TagsManager) Add(tag string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil
	}
	tags, err := t.Load()
	if err != nil {
		return err
	}
	// Case-insensitive duplicate check
	tagLower := strings.ToLower(tag)
	for _, existing := range tags {
		if strings.ToLower(existing) == tagLower {
			return nil // already exists
		}
	}
	tags = append(tags, tag)
	return t.save(tags)
}

// GetSuggestions returns tags whose prefix matches (case-insensitive).
func (t *TagsManager) GetSuggestions(prefix string) []string {
	tags, err := t.Load()
	if err != nil {
		return []string{}
	}
	if prefix == "" {
		return tags
	}
	prefixLower := strings.ToLower(prefix)
	var result []string
	for _, tag := range tags {
		if strings.HasPrefix(strings.ToLower(tag), prefixLower) {
			result = append(result, tag)
		}
	}
	return result
}
