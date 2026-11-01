package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	imagesPage = iota
	registriesPage
	archivesPage
	transfersPage
	preferencesPage
)

var pageNames = []string{"Images", "Registries", "Local archives", "Transfers", "Settings"}

// shell selects the approved Commander workspace for the native application.
func (desktop *Desktop) shell(version string) fyne.CanvasObject {
	return desktop.commanderShell(version)
}

// toolbar exposes navigation, command search, language and theme controls.
func (desktop *Desktop) toolbar() fyne.CanvasObject {
	brand := widget.NewLabelWithStyle("ImagePort", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	collapse := widget.NewButtonWithIcon("", theme.MenuIcon(), func() {
		if desktop.sidebar.Visible() {
			desktop.sidebar.Hide()
		} else {
			desktop.sidebar.Show()
		}
		desktop.window.Content().Refresh()
	})
	collapse.Importance = widget.LowImportance
	search := desktop.button("Search or jump to…", desktop.commandSearch)
	search.SetIcon(theme.SearchIcon())
	search.Importance = widget.LowImportance
	language := widget.NewButton("", func() {
		code := "zh-Hans"
		if desktop.language == code {
			code = "en"
		}
		desktop.changeLanguage(code)
	})
	desktop.bindText(func() {
		caption := "简体中文"
		if desktop.language == "zh-Hans" {
			caption = "English"
		}
		language.SetText(caption)
	})
	language.Importance = widget.LowImportance
	appearance := widget.NewButtonWithIcon("", theme.ColorPaletteIcon(), desktop.toggleTheme)
	appearance.Importance = widget.LowImportance
	left := container.NewHBox(collapse, brand, widget.NewSeparator(), desktop.breadcrumb)
	right := container.NewHBox(desktop.activity, language, appearance)
	return inset(container.NewBorder(nil, nil, left, right, container.NewCenter(search)), 10)
}

// navigation retains all five destinations and highlights the selected page.
func (desktop *Desktop) navigation(version string) fyne.CanvasObject {
	icons := []fyne.Resource{theme.StorageIcon(), theme.ComputerIcon(), theme.FolderIcon(), theme.DownloadIcon(), theme.SettingsIcon()}
	links := container.NewVBox(desktop.label("WORKSPACE"))
	for index, name := range pageNames {
		page := index
		button := desktop.button(name, func() { desktop.show(page) })
		button.SetIcon(icons[index])
		button.Alignment = widget.ButtonAlignLeading
		desktop.navigationButtons = append(desktop.navigationButtons, button)
		if index < preferencesPage {
			links.Add(button)
		}
	}
	help := desktop.button("Help & shortcuts", desktop.showHelp)
	help.SetIcon(theme.HelpIcon())
	help.Alignment = widget.ButtonAlignLeading
	help.Importance = widget.LowImportance
	bottom := container.NewVBox(widget.NewSeparator(), desktop.navigationButtons[preferencesPage], help, desktop.label("Local workspace"), widget.NewLabel(version))
	panel := inset(container.NewBorder(links, bottom, nil, nil, layout.NewSpacer()), 12)
	return container.New(&sidebarLayout{width: 208}, panel)
}

// show changes the page while preserving its form, filter and selection state.
func (desktop *Desktop) show(index int) {
	desktop.closeDrawer()
	if desktop.browser != nil && desktop.currentPage != index {
		desktop.browser.generation++
		desktop.localImages.generation++
	}
	desktop.currentPage = index
	desktop.body.Objects = []fyne.CanvasObject{desktop.pages[index]}
	desktop.body.Refresh()
	for page, button := range desktop.navigationButtons {
		button.Importance = widget.LowImportance
		if page == index {
			button.Importance = widget.HighImportance
		}
		button.Refresh()
	}
	desktop.breadcrumb.SetText(desktop.tr(pageNames[index]))
	if index == archivesPage {
		desktop.archives.refresh()
	}
	if index == imagesPage && desktop.copyView.sourceLocation.SelectedIndex() == 0 && !desktop.localImages.loaded {
		desktop.localImages.refresh()
	}
}

// updateActivity renders actual queued/running counts without implying transfer percentages.
func (desktop *Desktop) updateActivity() {
	if desktop.commander != nil {
		desktop.activity.SetText(desktop.tr("Transfers") + fmt.Sprintf(" · %d", len(desktop.tasks)))
		return
	}
	active := 0
	for _, item := range desktop.tasks {
		if item.State.Active() || item.State == "QUEUED" {
			active++
		}
	}
	desktop.activity.SetText(fmt.Sprintf("%d %s", active, desktop.tr("active")))
}

// toggleTheme persists the selected Fyne color variant for subsequent launches.
func (desktop *Desktop) toggleTheme() {
	dark := !desktop.app.Preferences().Bool("dark-theme")
	desktop.app.Preferences().SetBool("dark-theme", dark)
	desktop.restoreAppearance()
	if desktop.commander != nil {
		desktop.commander.activate(desktop.commander.active)
	}
}

// showHelp explains real supported navigation and image-management flows.
func (desktop *Desktop) showHelp() {
	desktop.inform("Help & shortcuts", "Choose a source location and image in Images, then choose a destination and copy. A destination tag names the copied image; no separate local tagging step is required. Remote sources require an explicit platform choice. Local archive destinations save OCI tar files. Transfers shows progress and verified results. Ctrl/Cmd+K opens search.")
}

// restoreAppearance applies the saved variant through Fyne's public theme API.
func (desktop *Desktop) restoreAppearance() {
	variant := theme.VariantLight
	if desktop.app.Preferences().Bool("dark-theme") {
		variant = theme.VariantDark
	}
	desktop.app.Settings().SetTheme(portTheme{Theme: theme.DefaultTheme(), variant: &variant})
}
