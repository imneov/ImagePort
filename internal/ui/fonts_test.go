package ui

import (
	"encoding/json"
	"os"
	"testing"
	"unicode"

	"fyne.io/fyne/v2"
	"golang.org/x/image/font/sfnt"
)

// TestInterfaceFontsCoverChineseCatalog prevents translated strings rendering as replacement glyphs.
func TestInterfaceFontsCoverChineseCatalog(test *testing.T) {
	data, err := os.ReadFile("../localization/zh-Hans.json")
	if err != nil {
		test.Fatal(err)
	}
	var messages map[string]string
	if err := json.Unmarshal(data, &messages); err != nil {
		test.Fatal(err)
	}
	for _, style := range []fyne.TextStyle{{}, {Bold: true}} {
		parsed, err := sfnt.Parse(Theme().Font(style).Content())
		if err != nil {
			test.Fatal(err)
		}
		for _, message := range messages {
			for _, character := range message {
				if !unicode.Is(unicode.Han, character) {
					continue
				}
				index, err := parsed.GlyphIndex(nil, character)
				if err != nil || index == 0 {
					test.Fatalf("missing glyph %q in style %+v: %v", character, style, err)
				}
			}
		}
	}
}
