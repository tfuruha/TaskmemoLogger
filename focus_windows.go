package main

// focus_windows.go
// Win32-level focus management for WebView2 embedded in Wails.
//
// Problem (root cause):
//   Wails' runtime.WindowShow() calls Win32 ShowWindow() on the top-level
//   Wails HWND, but this does NOT propagate keyboard focus into the WebView2
//   child control. As a result the OS keeps IME/keyboard attached to the
//   previously active window (or the desktop), causing the "ghost focus"
//   symptom where the WebView2 appears active but keystrokes go nowhere
//   (or get intercepted by the OS IME, showing characters in the screen
//   top-left corner).
//
// Solution:
//   After ShowWindow, enumerate the Wails top-level window's children and
//   call SetFocus() on the first HWND whose class name contains
//   "Chrome_WidgetWin_1" (the host window created by the Chromium engine
//   that backs WebView2). This bypasses the Wails/WebView2 initialization
//   race condition entirely at the Win32 message-loop level.

import (
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32dll   = windows.NewLazySystemDLL("user32.dll")
	kernel32dll = windows.NewLazySystemDLL("kernel32.dll")

	procFindWindowW        = user32dll.NewProc("FindWindowW")
	procEnumChildWindows   = user32dll.NewProc("EnumChildWindows")
	procGetClassNameW      = user32dll.NewProc("GetClassNameW")
	procSetFocus           = user32dll.NewProc("SetFocus")
	procSetForegroundWin   = user32dll.NewProc("SetForegroundWindow")
	procShowWindow         = user32dll.NewProc("ShowWindow")
	procAttachThreadInput  = user32dll.NewProc("AttachThreadInput")
	procGetWindowThreadPID = user32dll.NewProc("GetWindowThreadProcessId")
	// GetCurrentThreadId は kernel32.dll に存在する（user32.dll ではない）
	procGetCurrentThreadID = kernel32dll.NewProc("GetCurrentThreadId")

	// コールバックは一度だけ作成して再利用する（Goの仕様上、生成されたコールバックはGCされないため）
	enumChildWindowsCallback uintptr
	foundWebView2Child       uintptr
)

func init() {
	enumChildWindowsCallback = syscall.NewCallback(func(child, _ uintptr) uintptr {
		buf := make([]uint16, 256)
		procGetClassNameW.Call(child, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		className := windows.UTF16ToString(buf)
		if className == "Chrome_WidgetWin_1" {
			foundWebView2Child = child
			return 0 // stop enumeration
		}
		return 1 // continue
	})
}

const swShow = uintptr(5)

// forceWebView2Focus locates the Wails top-level window by its title,
// then finds the WebView2 child HWND and calls Win32 SetFocus() on it.
// This must be called from a goroutine (not the UI thread) after
// WindowShow() has returned.
func forceWebView2Focus(windowTitle string) {
	titlePtr, err := windows.UTF16PtrFromString(windowTitle)
	if err != nil {
		return
	}

	// Find the top-level Wails window.
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
	if hwnd == 0 {
		return
	}

	// NOTE: procShowWindow and procSetForegroundWin are redundant here
	// because wailsRuntime.WindowShow(ctx) was called just before this
	// asynchronous task started.

	// AttachThreadInput so our goroutine can call SetFocus().
	// Without this, SetFocus() from a different thread is silently ignored.
	fgThread, _, _ := procGetWindowThreadPID.Call(hwnd, 0)
	myThread, _, _ := procGetCurrentThreadID.Call()
	if fgThread != myThread {
		procAttachThreadInput.Call(fgThread, myThread, 1) // attach
		defer procAttachThreadInput.Call(fgThread, myThread, 0) // detach on return
	}

	// Enumerate child windows to find the WebView2 Chromium host.
	// We retry up to ~500 ms because Chrome_WidgetWin_1 may not exist yet
	// immediately after ShowWindow.
	// ポーリング間隔を 10ms に短縮し、即応性を向上させる。
	var target uintptr
	for range 50 {
		target = findWebView2Child(hwnd)
		if target != 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if target != 0 {
		procSetFocus.Call(target)
	}
}

// findWebView2Child enumerates children of hwnd and returns the first HWND
// whose Win32 class name is "Chrome_WidgetWin_1" (WebView2 host).
func findWebView2Child(hwnd uintptr) uintptr {
	foundWebView2Child = 0
	procEnumChildWindows.Call(hwnd, enumChildWindowsCallback, 0)
	return foundWebView2Child
}
