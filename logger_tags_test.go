package main

import (
	"fmt"
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

// ---- ReadRecent tests ----

func TestTaskLogger_ReadRecent_CurrentMonthOnly(t *testing.T) {
	tmpDir := t.TempDir()
	logger, _ := NewTaskLogger(tmpDir)

	now := time.Now()
	for i := 0; i < 5; i++ {
		_ = logger.Append(LogEntry{
			Timestamp: now.Format("2006-01-02 15:04"),
			Tags:      []string{"テスト"},
			Text:      fmt.Sprintf("エントリ%d", i+1),
		})
	}

	entries, err := logger.ReadRecent(recentLogLimit)
	if err != nil {
		t.Fatalf("ReadRecent: %v", err)
	}
	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}
}

func TestTaskLogger_ReadRecent_LimitApplied(t *testing.T) {
	tmpDir := t.TempDir()
	logger, _ := NewTaskLogger(tmpDir)

	now := time.Now()
	for i := 0; i < 25; i++ {
		_ = logger.Append(LogEntry{
			Timestamp: now.Format("2006-01-02 15:04"),
			Tags:      nil,
			Text:      fmt.Sprintf("エントリ%d", i+1),
		})
	}

	entries, err := logger.ReadRecent(recentLogLimit)
	if err != nil {
		t.Fatalf("ReadRecent: %v", err)
	}
	if len(entries) != recentLogLimit {
		t.Errorf("expected %d entries, got %d", recentLogLimit, len(entries))
	}
	// 最新 recentLogLimit 件であることを確認（最後のエントリのテキストで検証）
	last := entries[len(entries)-1]
	if last.Text != fmt.Sprintf("エントリ%d", 25) {
		t.Errorf("last entry should be エントリ25, got %q", last.Text)
	}
}

func TestTaskLogger_ReadRecent_CrossMonth(t *testing.T) {
	tmpDir := t.TempDir()
	logger, _ := NewTaskLogger(tmpDir)

	now := time.Now()
	prevMonth := now.AddDate(0, -1, 0)

	// 前月ファイルに直接３件書き込む
	prevPath := logger.logFilePathFor(prevMonth)
	for i := 0; i < 3; i++ {
		_ = appendToPath(prevPath, prevMonth.Format("2006-01-02 15:04"), fmt.Sprintf("前月エントリ%d", i+1))
	}
	// 今月に5件
	for i := 0; i < 5; i++ {
		_ = logger.Append(LogEntry{
			Timestamp: now.Format("2006-01-02 15:04"),
			Tags:      nil,
			Text:      fmt.Sprintf("今月エントリ%d", i+1),
		})
	}

	entries, err := logger.ReadRecent(recentLogLimit)
	if err != nil {
		t.Fatalf("ReadRecent: %v", err)
	}
	// 合計8件（20件未満なので全件返る）
	if len(entries) != 8 {
		t.Errorf("expected 8 entries (3 prev + 5 current), got %d", len(entries))
	}
	// 先頭は前月、末尾は今月
	if entries[0].Text != "前月エントリ1" {
		t.Errorf("first entry should be 前月エントリ1, got %q", entries[0].Text)
	}
	if entries[7].Text != "今月エントリ5" {
		t.Errorf("last entry should be 今月エントリ5, got %q", entries[7].Text)
	}
}

func TestTaskLogger_ReadRecent_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	logger, _ := NewTaskLogger(tmpDir)

	entries, err := logger.ReadRecent(recentLogLimit)
	if err != nil {
		t.Fatalf("ReadRecent on empty dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

// appendToPath は指定パスへ直接エントリを書き込むテスト用ヘルパー。
func appendToPath(path, timestamp, text string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	block := fmt.Sprintf("\n## %s\n- %s\n", timestamp, text)
	_, err = f.WriteString(block)
	return err
}

// ---- App tests ----

func TestApp_SaveLogWithOffset(t *testing.T) {
	tmpDir := t.TempDir()
	logger, _ := NewTaskLogger(tmpDir)
	tagsMgr, _ := NewTagsManager(tmpDir)

	app := &App{
		logger:      logger,
		tagsManager: tagsMgr,
	}

	// 15分前のオフセットで保存
	err := app.SaveLog([]string{"テスト"}, "オフセットテスト", -15)
	if err != nil {
		t.Fatalf("SaveLog: %v", err)
	}

	entries, err := logger.ReadToday()
	if err != nil {
		t.Fatalf("ReadToday: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	expectedTime := time.Now().Add(-15 * time.Minute).Format("2006-01-02 15:04")
	if entries[0].Timestamp != expectedTime {
		t.Errorf("timestamp mismatch: want %q, got %q", expectedTime, entries[0].Timestamp)
	}
}

func TestApp_SaveWorkStart(t *testing.T) {
	tmpDir := t.TempDir()
	logger, _ := NewTaskLogger(tmpDir)
	tagsMgr, _ := NewTagsManager(tmpDir)

	app := &App{
		logger:      logger,
		tagsManager: tagsMgr,
	}

	// 30分前のオフセットで始業保存
	err := app.SaveWorkStart(-30)
	if err != nil {
		t.Fatalf("SaveWorkStart: %v", err)
	}

	entries, err := logger.ReadToday()
	if err != nil {
		t.Fatalf("ReadToday: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	expectedTime := time.Now().Add(-30 * time.Minute).Format("2006-01-02 15:04")
	if entries[0].Timestamp != expectedTime {
		t.Errorf("timestamp mismatch: want %q, got %q", expectedTime, entries[0].Timestamp)
	}
	if len(entries[0].Tags) != 1 || entries[0].Tags[0] != "始業" {
		t.Errorf("tags mismatch: want [始業], got %v", entries[0].Tags)
	}
	if entries[0].Text != "始業" {
		t.Errorf("text mismatch: want %q, got %q", "始業", entries[0].Text)
	}
}

func TestApp_HasWorkStartToday(t *testing.T) {
	tmpDir := t.TempDir()
	logger, _ := NewTaskLogger(tmpDir)
	tagsMgr, _ := NewTagsManager(tmpDir)

	app := &App{
		logger:      logger,
		tagsManager: tagsMgr,
	}

	// 初期状態は false であること
	hasStart, err := app.HasWorkStartToday()
	if err != nil {
		t.Fatalf("HasWorkStartToday failed: %v", err)
	}
	if hasStart {
		t.Error("expected false, got true")
	}

	// 始業を保存
	err = app.SaveWorkStart(0)
	if err != nil {
		t.Fatalf("SaveWorkStart failed: %v", err)
	}

	// 始業保存後は true になること
	hasStart, err = app.HasWorkStartToday()
	if err != nil {
		t.Fatalf("HasWorkStartToday failed: %v", err)
	}
	if !hasStart {
		t.Error("expected true, got false")
	}
}

