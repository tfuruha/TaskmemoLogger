package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the main application struct. Public methods are exposed to the frontend.
type App struct {
	ctx         context.Context
	logger      *TaskLogger
	tagsManager *TagsManager
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. Initialises logger and tagsManager.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home dir:", err)
		return
	}

	// Log directory: %USERPROFILE%\Documents\TaskmemoLogger\log
	logDir := filepath.Join(home, "Documents", "TaskmemoLogger", "log")
	a.logger, err = NewTaskLogger(logDir)
	if err != nil {
		fmt.Println("Error initialising logger:", err)
	}

	// Tags data directory: %USERPROFILE%\Documents\TaskmemoLogger
	tagsDir := filepath.Join(home, "Documents", "TaskmemoLogger")
	a.tagsManager, err = NewTagsManager(tagsDir)
	if err != nil {
		fmt.Println("Error initialising tags manager:", err)
	}
}

// domReady is called after the frontend DOM has been loaded.
// It shows the window and forces Win32-level keyboard focus into the
// WebView2 child control, then signals the frontend to focus the input.
func (a *App) domReady(ctx context.Context) {
	// 1. Show the window at the OS level.
	wailsRuntime.WindowShow(ctx)

	// 2. Force Win32 keyboard focus directly onto the WebView2 HWND.
	//    This resolves the "ghost focus" issue where the embedded Chrome
	//    engine doesn't receive OS-level keyboard input after startup.
	forceWebView2Focus("TaskmemoLogger")

	// 3. Signal frontend to call element.focus() now that the OS-level
	//    focus is correctly on the WebView2 control.
	wailsRuntime.EventsEmit(ctx, "app:ready-focus")
}

// SaveLog saves a task entry to the Markdown log file and persists any new tags.
// This is called by the frontend on submit.
func (a *App) SaveLog(tags []string, text string) error {
	if a.logger == nil || a.tagsManager == nil {
		return fmt.Errorf("app not initialised")
	}
	entry := LogEntry{
		Timestamp: time.Now().Format("2006-01-02 15:04"),
		Tags:      tags,
		Text:      text,
	}
	// 1. Write to Markdown file
	if err := a.logger.Append(entry); err != nil {
		return fmt.Errorf("failed to save log: %w", err)
	}
	// 2. Persist new tags
	for _, tag := range tags {
		if err := a.tagsManager.Add(tag); err != nil {
			// Non-fatal: log but continue
			fmt.Println("Error saving tag:", tag, err)
		}
	}
	return nil
}

// GetTodayLogs returns today's log entries for the chat history view.
func (a *App) GetTodayLogs() ([]LogEntry, error) {
	if a.logger == nil {
		return nil, fmt.Errorf("app not initialised")
	}
	return a.logger.ReadToday()
}

// GetTagSuggestions returns tags matching the given prefix.
func (a *App) GetTagSuggestions(prefix string) []string {
	if a.tagsManager == nil {
		return []string{}
	}
	return a.tagsManager.GetSuggestions(prefix)
}

// AddTag adds a tag to the persistent tags list.
func (a *App) AddTag(tag string) error {
	if a.tagsManager == nil {
		return fmt.Errorf("app not initialised")
	}
	return a.tagsManager.Add(tag)
}
