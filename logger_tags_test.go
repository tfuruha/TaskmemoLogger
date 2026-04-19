package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---- TaskLogger tests ----

func TestTaskLogger_AppendAndReadToday(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewTaskLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewTaskLogger: %v", err)
	}

	now := time.Now()
	entry := LogEntry{
		Timestamp: now.Format("2006-01-02 15:04"),
		Tags:      []string{"開発", "テスト"},
		Text:      "ユニットテストを書いた",
	}

	if err := logger.Append(entry); err != nil {
		t.Fatalf("Append: %v", err)
	}

	entries, err := logger.ReadToday()
	if err != nil {
		t.Fatalf("ReadToday: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got := entries[0]
	if got.Timestamp != entry.Timestamp {
		t.Errorf("timestamp: want %q, got %q", entry.Timestamp, got.Timestamp)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "開発" || got.Tags[1] != "テスト" {
		t.Errorf("tags: want %v, got %v", entry.Tags, got.Tags)
	}
	if got.Text != entry.Text {
		t.Errorf("text: want %q, got %q", entry.Text, got.Text)
	}
}

func TestTaskLogger_MultilineText(t *testing.T) {
	tmpDir := t.TempDir()
	logger, _ := NewTaskLogger(tmpDir)

	entry := LogEntry{
		Timestamp: time.Now().Format("2006-01-02 15:04"),
		Tags:      []string{"会議"},
		Text:      "1行目\n2行目\n3行目",
	}
	if err := logger.Append(entry); err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Verify file contains all lines
	month := time.Now().Format("2006-01")
	data, _ := os.ReadFile(filepath.Join(tmpDir, month+"_log.md"))
	content := string(data)
	if !strings.Contains(content, "1行目") || !strings.Contains(content, "2行目") {
		t.Errorf("multiline text not found in file:\n%s", content)
	}
}

func TestTaskLogger_EmptyLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	logger, _ := NewTaskLogger(tmpDir)

	entries, err := logger.ReadToday()
	if err != nil {
		t.Fatalf("ReadToday on empty dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

// ---- TagsManager tests ----

func TestTagsManager_AddAndSuggest(t *testing.T) {
	tmpDir := t.TempDir()
	tm, err := NewTagsManager(tmpDir)
	if err != nil {
		t.Fatalf("NewTagsManager: %v", err)
	}

	for _, tag := range []string{"開発", "会議", "ドキュメント"} {
		if err := tm.Add(tag); err != nil {
			t.Fatalf("Add(%q): %v", tag, err)
		}
	}

	tests := []struct {
		prefix string
		want   []string
	}{
		{"開", []string{"開発"}},
		{"会", []string{"会議"}},
		{"ド", []string{"ドキュメント"}},
		{"", []string{"開発", "会議", "ドキュメント"}},
		{"存在しない", []string{}},
	}
	for _, tc := range tests {
		got := tm.GetSuggestions(tc.prefix)
		if len(got) != len(tc.want) {
			t.Errorf("GetSuggestions(%q): want %v, got %v", tc.prefix, tc.want, got)
		}
	}
}

func TestTagsManager_DuplicatePrevention(t *testing.T) {
	tmpDir := t.TempDir()
	tm, _ := NewTagsManager(tmpDir)

	_ = tm.Add("開発")
	_ = tm.Add("開発") // duplicate
	_ = tm.Add("開発") // duplicate

	tags, _ := tm.Load()
	if len(tags) != 1 {
		t.Errorf("expected 1 tag after duplicate adds, got %d: %v", len(tags), tags)
	}
}

func TestTagsManager_CaseInsensitiveDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	tm, _ := NewTagsManager(tmpDir)

	_ = tm.Add("Meeting")
	_ = tm.Add("meeting") // case-insensitive duplicate
	_ = tm.Add("MEETING") // case-insensitive duplicate

	tags, _ := tm.Load()
	if len(tags) != 1 {
		t.Errorf("expected 1 tag, got %d: %v", len(tags), tags)
	}
}

func TestTagsManager_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	tm, _ := NewTagsManager(tmpDir)

	_ = tm.Add("タグA")
	_ = tm.Add("タグB")

	// Simulate re-reading from disk (as if reloaded)
	tm2, _ := NewTagsManager(tmpDir)
	tags, _ := tm2.Load()
	if len(tags) != 2 {
		t.Errorf("expected 2 tags after reload, got %d", len(tags))
	}
}

func TestTagsManager_EmptyTag(t *testing.T) {
	tmpDir := t.TempDir()
	tm, _ := NewTagsManager(tmpDir)

	if err := tm.Add(""); err != nil {
		t.Errorf("Add empty string should not error: %v", err)
	}
	tags, _ := tm.Load()
	if len(tags) != 0 {
		t.Errorf("empty tag should not be saved, got %v", tags)
	}
}
