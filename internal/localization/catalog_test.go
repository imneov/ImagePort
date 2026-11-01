package localization

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// TestLanguageSelection checks default/fallback policy and formatted application messages.
func TestLanguageSelection(test *testing.T) {
	catalog, err := New("en")
	if err != nil {
		test.Fatal(err)
	}
	if catalog.Text("Settings") != "Settings" {
		test.Fatal("English is not the default")
	}
	if err := catalog.SetLanguage("zh-Hans"); err != nil {
		test.Fatal(err)
	}
	if catalog.Text("Settings") != "设置" {
		test.Fatal("Chinese selection not applied")
	}
	message := catalog.Text("Found %d entries; select one. All platforms in the selected entry will be preserved.", 3)
	if !strings.Contains(message, "3 个条目") {
		test.Fatal("format arguments lost")
	}
	if err := catalog.SetLanguage("unsupported"); err == nil || catalog.Text("Settings") != "设置" {
		test.Fatal("invalid selection changed language")
	}
	if catalog.Text("Unknown upstream diagnostic") != "Unknown upstream diagnostic" {
		test.Fatal("unknown diagnostics lost")
	}
	fallback, err := New("unsupported")
	if err != nil || fallback.Text("Settings") != "Settings" {
		test.Fatal("invalid saved preference did not fall back to English")
	}
}

// TestLiteralDiagnostics preserves filenames and upstream text containing template syntax.
func TestLiteralDiagnostics(test *testing.T) {
	for _, code := range []string{"en", "zh-Hans"} {
		catalog, err := New(code)
		if err != nil {
			test.Fatal(err)
		}
		for _, message := range []string{`/tmp/{{printf "example"}}.tar`, `unexpected {{.Value}} at 50%`, `invalid {{ syntax`} {
			if actual := catalog.Text(message); actual != message {
				test.Errorf("%s altered diagnostic: %q", code, actual)
			}
		}
	}
}

// TestCatalogFormats rejects translations that change interpolation verbs or add missing args.
func TestCatalogFormats(test *testing.T) {
	var messages map[string]string
	if err := json.Unmarshal(chinese, &messages); err != nil {
		test.Fatal(err)
	}
	for english, translated := range messages {
		if formatVerbs(english) != formatVerbs(translated) {
			test.Errorf("format changed for %q", english)
		}
		if translated == "" {
			test.Errorf("empty translation for %q", english)
		}
	}
}

// formatVerbs returns the printf directives in message for catalog compatibility checks.
func formatVerbs(message string) string {
	var directives []string
	for index := 0; index < len(message); index++ {
		if message[index] != '%' {
			continue
		}
		end := index + 1
		for end < len(message) && !strings.ContainsRune("vTtbcdoOqxXUeEfFgGspw%", rune(message[end])) {
			end++
		}
		if end < len(message) {
			directives = append(directives, message[index:end+1])
			index = end
		}
	}
	return fmt.Sprint(directives)
}

// TestConcurrentSelection exercises background translation during repeated language switches.
func TestConcurrentSelection(test *testing.T) {
	catalog, err := New("en")
	if err != nil {
		test.Fatal(err)
	}
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		for range 100 {
			catalog.Text("Ready")
		}
	}()
	for range 100 {
		if err := catalog.SetLanguage("zh-Hans"); err != nil {
			test.Fatal(err)
		}
		if err := catalog.SetLanguage("en"); err != nil {
			test.Fatal(err)
		}
	}
	workers.Wait()
}
