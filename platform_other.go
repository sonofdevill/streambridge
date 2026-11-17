//go:build !windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
)

func firewallSupported() bool { return false }

func fixFirewall() error { return errors.New("only needed on Windows") }

func autostartSupported() bool      { return false }
func autostartEnabled() bool        { return false }
func setAutostart(on bool) error    { return errors.New("only available on Windows") }
func refreshAutostartPath()         {}
func setupLogging(cfgPath string)   {}

func showFatal(msg string) {
	fmt.Println("  ERROR: " + msg)
	fmt.Println("  Press Enter to close.")
	var s string
	_, _ = fmt.Scanln(&s)
}

// runUI waits for Ctrl+C or a quit request from the control panel.
func runUI(app *App) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	select {
	case <-sig:
	case <-app.quit:
	}
}

func openBrowser(url string) {
	if runtime.GOOS == "darwin" {
		_ = exec.Command("open", url).Start()
		return
	}
	// When started with sudo, open the browser as the real user.
	if u := os.Getenv("SUDO_USER"); u != "" {
		_ = exec.Command("sudo", "-u", u, "xdg-open", url).Start()
		return
	}
	_ = exec.Command("xdg-open", url).Start()
}
