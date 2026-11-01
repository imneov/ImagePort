// ImagePort is a daemonless graphical OCI image management client.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/gofrs/flock"
	dockeradapter "github.com/imneov/ImagePort/internal/adapter/docker"
	adapter "github.com/imneov/ImagePort/internal/adapter/regclient"
	"github.com/imneov/ImagePort/internal/archive"
	"github.com/imneov/ImagePort/internal/presenter"
	"github.com/imneov/ImagePort/internal/secret"
	"github.com/imneov/ImagePort/internal/store"
	"github.com/imneov/ImagePort/internal/task"
	"github.com/imneov/ImagePort/internal/transfer"
	"github.com/imneov/ImagePort/internal/ui"
)

var version = "0.1.0-dev"

// main reports initialization failures without printing secret-bearing SDK errors.
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "ImagePort:", err)
		os.Exit(1)
	}
}

// run initializes a single-instance desktop and tears resources down in reverse order.
func run() error {
	directory := flag.String("data-dir", "", "metadata directory (defaults to the OS configuration directory)")
	smoke := flag.Bool("smoke", false, "open and capture the desktop, then exit (use -tags ci without a display)")
	screenshot := flag.String("screenshot", "", "optional PNG path for smoke verification")
	flag.Parse()
	path, err := dataPath(*directory)
	if err != nil {
		return err
	}
	lock := flock.New(filepath.Join(path, "instance.lock"))
	locked, err := lock.TryLock()
	if err != nil {
		return fmt.Errorf("lock application data directory: %w", err)
	}
	if !locked {
		return errors.New("Another ImagePort instance is using this data directory")
	}
	defer func() {
		if err := lock.Unlock(); err != nil {
			slog.Error("instance lock release failed")
		}
	}()
	database, err := store.Open(filepath.Join(path, "imageport.db"))
	if err != nil {
		return err
	}
	defer closeDatabase(database)
	application := newApplication()
	application.Settings().SetTheme(ui.Theme())
	vault := secret.New()
	port := adapter.New(vault)
	runner := &transfer.Runner{Registry: port, Local: dockeradapter.New(), Profiles: database, Limits: archive.DefaultLimits(), Logger: slog.Default()}
	manager, err := task.New(database, runner, application.Preferences().IntWithFallback("concurrency", 2), slog.Default())
	if err != nil {
		return err
	}
	service := presenter.New(database, vault, port, manager)
	defer service.Close()
	defer shutdown(manager)
	return launch(application, service, *smoke, *screenshot)
}

// closeDatabase reports teardown failure without exposing metadata or credentials.
func closeDatabase(database *store.Store) {
	if err := database.Close(); err != nil {
		slog.Error("database close failed")
	}
}

// dataPath creates an application-specific metadata directory without privileges.
func dataPath(override string) (string, error) {
	if override == "" {
		root, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		override = filepath.Join(root, "ImagePort")
	}
	if err := os.MkdirAll(override, 0700); err != nil {
		return "", err
	}
	return filepath.Abs(override)
}

// shutdown keeps database access alive until task workers have stopped.
func shutdown(manager *task.Manager) {
	if err := manager.Close(context.Background()); err != nil {
		slog.Error("task shutdown failed")
	}
}

// capture exercises actual Fyne rendering without requiring any remote service.
func capture(window fyne.Window, filename string) error {
	window.Show()
	if filename == "" {
		return nil
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	err = png.Encode(file, window.Canvas().Capture())
	return errors.Join(err, file.Close())
}

// launch opens the native event loop or performs an explicit headless rendering check.
func launch(application fyne.App, service *presenter.Presenter, smoke bool, screenshot string) error {
	window := application.NewWindow("ImagePort · Image Management")
	window.SetIcon(theme.StorageIcon())
	if _, err := ui.New(application, window, service, version); err != nil {
		return err
	}
	if smoke {
		if err := waitForSmoke(service); err != nil {
			return err
		}
		return capture(window, screenshot)
	}
	window.ShowAndRun()
	return nil
}
