package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// accountEditor retains form state after validation and keyring failures.
type accountEditor struct {
	workspace                    *commanderView
	target                       domain.Target
	existing                     domain.Profile
	name, username, password, ca *widget.Entry
	session                      *widget.Check
	feedback                     *widget.Label
	save                         *widget.Button
	popup                        *dialog.CustomDialog
	closed                       bool
	saved                        func(domain.Profile)
}

// accountDialog edits one identity bound to the selected registry, without editing its address.
func (view *commanderView) accountDialog(target domain.Target, existing domain.Profile, saved func(domain.Profile)) {
	editor := &accountEditor{workspace: view, target: target, existing: existing, saved: saved}
	editor.name, editor.username = widget.NewEntry(), widget.NewEntry()
	editor.password, editor.ca = widget.NewPasswordEntry(), widget.NewEntry()
	editor.name.SetText(existing.Name)
	editor.username.SetText(existing.Username)
	editor.password.SetPlaceHolder(view.desktop.tr("Leave blank to keep existing credentials"))
	editor.ca.SetPlaceHolder(view.desktop.tr("Optional PEM CA certificate file"))
	editor.session = widget.NewCheck(view.desktop.tr("Keep credentials for this session only"), nil)
	editor.session.SetChecked(existing.SessionOnly)
	editor.feedback = widget.NewLabel("")
	editor.feedback.Wrapping = fyne.TextWrapWord
	editor.feedback.Hide()
	fields := widget.NewForm(
		widget.NewFormItem(view.desktop.tr("Registry address"), widget.NewLabel(target.Address)),
		widget.NewFormItem(view.desktop.tr("Display name"), editor.name),
		widget.NewFormItem(view.desktop.tr("Username"), editor.username),
		widget.NewFormItem(view.desktop.tr("Password / token"), editor.password),
		widget.NewFormItem(view.desktop.tr("CA file"), editor.ca),
		widget.NewFormItem(view.desktop.tr("Credentials"), editor.session))
	editor.save = widget.NewButton(view.desktop.tr("Save account"), editor.submit)
	editor.save.Importance = widget.HighImportance
	cancel := widget.NewButton(view.desktop.tr("Cancel"), func() { editor.popup.Hide() })
	editor.popup = dialog.NewCustomWithoutButtons(view.desktop.tr("Access account"), container.NewVBox(fields, editor.feedback, container.NewHBox(cancel, editor.save)), view.desktop.window)
	editor.popup.SetOnClosed(func() { editor.closed = true; editor.password.SetText("") })
	editor.popup.Resize(fyne.NewSize(700, 450))
	editor.popup.Show()
}

// submit captures metadata on the UI thread and stores secrets through the presenter's vault.
func (editor *accountEditor) submit() {
	parsed, err := url.Parse(editor.target.Address)
	if err != nil {
		editor.fail(err)
		return
	}
	profile := editor.existing
	profile.TargetID, profile.Host, profile.HTTP = editor.target.ID, parsed.Host, parsed.Scheme == "http"
	profile.Name, profile.Username = strings.TrimSpace(editor.name.Text), strings.TrimSpace(editor.username.Text)
	profile.SessionOnly = editor.session.Checked
	if profile.Username == "" {
		editor.fail(errors.New("Enter an account username"))
		return
	}
	password, caPath := editor.password.Text, strings.TrimSpace(editor.ca.Text)
	editor.password.SetText("")
	editor.save.Disable()
	editor.workspace.desktop.presenter.Work(func(context.Context) (any, error) {
		if caPath != "" {
			data, err := readAccountCA(caPath)
			if err != nil {
				return nil, err
			}
			profile.CAPEM = data
		}
		return editor.workspace.desktop.presenter.SaveProfile(profile, password)
	}, func(result any, err error) {
		fyne.Do(func() {
			if editor.closed {
				return
			}
			editor.save.Enable()
			if err != nil {
				editor.fail(err)
				return
			}
			profile := result.(domain.Profile)
			editor.workspace.upsertAccount(profile)
			editor.popup.Hide()
			if editor.saved != nil {
				editor.saved(profile)
			}
		})
	})
}

// fail displays an actionable error while keeping entered account metadata available.
func (editor *accountEditor) fail(err error) {
	editor.feedback.SetText(editor.workspace.desktop.translateDetails(err.Error()))
	editor.feedback.Show()
}

// readAccountCA reads a bounded certificate file; it returns PEM text or a contextual error.
func readAccountCA(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("read CA certificate: %w", err)
	}
	data, readErr := io.ReadAll(io.LimitReader(file, 1<<20+1))
	if err := errors.Join(readErr, file.Close()); err != nil {
		return "", err
	}
	if len(data) > 1<<20 {
		return "", errors.New("CA certificate file exceeds 1 MiB")
	}
	return string(data), nil
}

// upsertAccount publishes saved metadata before an open copy dialog reloads its choices.
func (view *commanderView) upsertAccount(profile domain.Profile) {
	for index, current := range view.accounts {
		if current.ID == profile.ID {
			view.accounts[index] = profile
			return
		}
	}
	view.accounts = append(view.accounts, profile)
}
