package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// LogEntry represents a single task log record.
type LogEntry struct {
	Timestamp string   `json:"timestamp"`
	Tags      []string `json:"tags"`
	Text      string   `json:"text"`
}

// TaskLogger handles file I/O for Markdown task logs.
type TaskLogger struct {
	logDir string
}

// NewTaskLogger creates a TaskLogger. logDir is %USERPROFILE%\Documents\TaskmemoLogger\log.
func NewTaskLogger(logDir string) (*TaskLogger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}
	return &TaskLogger{logDir: logDir}, nil
}

// logFilePath returns the current month's log file path.
func (l *TaskLogger) logFilePath() string {
	month := time.Now().Format("2006-01")
	return filepath.Join(l.logDir, month+"_log.md")
}

// Append writes a new LogEntry to the Markdown log file.
// Format (compatible with Python version):
//
//	## 2026-04-18 15:30
//	- [タグ1] [タグ2]
//	- タスク内容
func (l *TaskLogger) Append(entry LogEntry) error {
	f, err := os.OpenFile(l.logFilePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	// Build tag string: [タグ1] [タグ2]
	tagLine := ""
	for _, t := range entry.Tags {
		tagLine += fmt.Sprintf("[%s] ", t)
	}
	tagLine = strings.TrimRight(tagLine, " ")

	// Indent multi-line text content
	lines := strings.Split(strings.TrimRight(entry.Text, "\n"), "\n")
	textLine := lines[0]
	extra := ""
	for _, l := range lines[1:] {
		extra += "  " + l + "\n"
	}

	block := fmt.Sprintf("\n## %s\n", entry.Timestamp)
	if tagLine != "" {
		block += fmt.Sprintf("- %s\n", tagLine)
	}
	block += fmt.Sprintf("- %s\n", textLine)
	if extra != "" {
		block += extra
	}

	_, err = f.WriteString(block)
	return err
}

// ReadToday parses today's entries from the current month's log file.
func (l *TaskLogger) ReadToday() ([]LogEntry, error) {
	today := time.Now().Format("2006-01-02")
	path := l.logFilePath()

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []LogEntry{}, nil
		}
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	var entries []LogEntry
	var current *LogEntry
	var bodyLines []string

	headerRe := regexp.MustCompile(`^## (\d{4}-\d{2}-\d{2} \d{2}:\d{2})`)
	tagLineRe := regexp.MustCompile(`^\- \[`)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if m := headerRe.FindStringSubmatch(line); m != nil {
			// Save previous entry
			if current != nil {
				current.Text = strings.TrimRight(strings.Join(bodyLines, "\n"), "\n")
				if strings.HasPrefix(current.Timestamp, today) {
					entries = append(entries, *current)
				}
			}
			current = &LogEntry{Timestamp: m[1]}
			bodyLines = nil
			continue
		}
		if current == nil {
			continue
		}
		if tagLineRe.MatchString(line) {
			// Parse tags: - [タグ1] [タグ2]
			tagRe := regexp.MustCompile(`\[([^\]]+)\]`)
			matches := tagRe.FindAllStringSubmatch(line, -1)
			for _, tm := range matches {
				current.Tags = append(current.Tags, tm[1])
			}
			continue
		}
		// Body line (strip leading "- ")
		if strings.HasPrefix(line, "- ") {
			bodyLines = append(bodyLines, strings.TrimPrefix(line, "- "))
		} else if strings.HasPrefix(line, "  ") {
			bodyLines = append(bodyLines, strings.TrimPrefix(line, "  "))
		} else if line != "" {
			bodyLines = append(bodyLines, line)
		}
	}
	// Save last entry
	if current != nil {
		current.Text = strings.TrimRight(strings.Join(bodyLines, "\n"), "\n")
		if strings.HasPrefix(current.Timestamp, today) {
			entries = append(entries, *current)
		}
	}

	return entries, scanner.Err()
}
