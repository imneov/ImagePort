// Package localization provides application translations independently of OS locale.
package localization

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/nicksnyder/go-i18n/v2/i18n/template"
	"golang.org/x/text/language"
)

//go:embed zh-Hans.json
var chinese []byte

// Catalog owns immutable messages and an atomically replaceable language selection.
type Catalog struct {
	bundle *i18n.Bundle
	active atomic.Pointer[i18n.Localizer]
}

// New loads bundled translations and selects code, falling back to English if invalid.
func New(code string) (*Catalog, error) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	if _, err := bundle.ParseMessageFileBytes(chinese, "zh-Hans.json"); err != nil {
		return nil, fmt.Errorf("load application translations: %w", err)
	}
	catalog := &Catalog{bundle: bundle}
	if err := catalog.SetLanguage(code); err != nil {
		if err := catalog.SetLanguage("en"); err != nil {
			return nil, err
		}
	}
	return catalog, nil
}

// SetLanguage selects a supported BCP 47 code; invalid codes leave selection intact.
func (catalog *Catalog) SetLanguage(code string) error {
	if code != "en" && code != "zh-Hans" {
		return errors.New("Unsupported interface language")
	}
	catalog.active.Store(i18n.NewLocalizer(catalog.bundle, code, "en"))
	return nil
}

// Text translates an English message template and formats its optional arguments.
func (catalog *Catalog) Text(message string, arguments ...any) string {
	text, err := catalog.active.Load().Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{ID: message, Other: message},
		// Only printf interpolation is supported; diagnostics and paths stay literal.
		TemplateParser: template.IdentityParser{},
	})
	if err != nil && text == "" {
		text = message
	}
	if len(arguments) > 0 {
		return fmt.Sprintf(text, arguments...)
	}
	return text
}
