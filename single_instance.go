package main

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

// mutexName is a well-known name for the Windows named mutex.
// If another instance is already running, CreateMutex returns ERROR_ALREADY_EXISTS.
const mutexName = "Local\\TaskmemoLogger_SingleInstance"

// acquireSingleInstanceLock attempts to create a named Windows mutex.
// Returns the mutex handle (must be released on exit) and nil when this is
// the first instance. Returns nil handle + nil error when a second instance
// is detected (caller should exit immediately).
// Returns non-nil error only on unexpected failures.
func acquireSingleInstanceLock() (windows.Handle, error) {
	name, err := windows.UTF16PtrFromString(mutexName)
	if err != nil {
		return 0, fmt.Errorf("failed to encode mutex name: %w", err)
	}

	handle, err := windows.CreateMutex(nil, false, name)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		// Another instance is already running.
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to create mutex: %w", err)
	}

	return handle, nil
}
