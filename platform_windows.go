//go:build windows

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

func firewallSupported() bool { return true }

// fixFirewall adds an inbound allow rule for this program (DNS 53, RTMP 1935,
// phone control), asking for administrator approval through the normal UAC prompt.
func fixFirewall() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	q := strings.ReplaceAll(exe, "'", "''")
	ps := fmt.Sprintf("Remove-NetFirewallRule -DisplayName 'StreamBridge' -ErrorAction SilentlyContinue; "+
		"New-NetFirewallRule -DisplayName 'StreamBridge' -Direction Inbound -Program '%s' -Action Allow -Profile Any | Out-Null", q)
	params := `-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -Command "` + ps + `"`

	shell32 := syscall.NewLazyDLL("shell32.dll")
	proc := shell32.NewProc("ShellExecuteW")
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString("powershell.exe")
	args, _ := syscall.UTF16PtrFromString(params)
	r, _, _ := proc.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), uintptr(unsafe.Pointer(args)), 0, 0)
	if r <= 32 {
		return fmt.Errorf("administrator approval was not given (code %d)", r)
	}
	return nil
}

func openBrowser(url string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

// ---- Start with Windows (per-user, no admin needed)

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
const runValueName = "StreamBridge"

func autostartSupported() bool { return true }

func autostartCommand() string {
	exe, _ := os.Executable()
	return `"` + exe + `" -background`
}

func autostartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetStringValue(runValueName)
	return err == nil && v != ""
}

func setAutostart(on bool) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if on {
		return k.SetStringValue(runValueName, autostartCommand())
	}
	err = k.DeleteValue(runValueName)
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}

// refreshAutostartPath keeps the startup entry pointing at this exe if the
// user moved the file.
func refreshAutostartPath() {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()
	v, _, err := k.GetStringValue(runValueName)
	if err == nil && v != "" && v != autostartCommand() {
		_ = k.SetStringValue(runValueName, autostartCommand())
	}
}

// ---- Logging and errors (the Windows build has no console window)

func setupLogging(cfgPath string) {
	dir := filepath.Dir(cfgPath)
	_ = os.MkdirAll(dir, 0o700)
	p := filepath.Join(dir, "streambridge.log")
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if st, err := os.Stat(p); err == nil && st.Size() > 2<<20 {
		flags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}
	f, err := os.OpenFile(p, flags, 0o600)
	if err != nil {
		return
	}
	log.SetOutput(io.MultiWriter(f, os.Stderr))
}

func showFatal(msg string) {
	log.Println("FATAL: " + msg)
	user32 := syscall.NewLazyDLL("user32.dll")
	box := user32.NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(msg)
	c, _ := syscall.UTF16PtrFromString("StreamBridge")
	_, _, _ = box.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), 0x10) // MB_ICONERROR
}
