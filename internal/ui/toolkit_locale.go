package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"sync"
)

var toolkitLocale struct {
	once sync.Once
	err  error
}

// registerToolkitLocale fills the toolkit's missing Chinese select prompt before creating widgets.
func registerToolkitLocale() error {
	toolkitLocale.once.Do(func() {
		toolkitLocale.err = lang.AddTranslationsForLocale([]byte(`{"(Select one)":"请选择"}`), fyne.Locale("zh-Hans"))
	})
	return toolkitLocale.err
}
