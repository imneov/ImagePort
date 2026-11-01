package ui

import (
	"context"
	"strings"

	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// localTagDialog requests an explicit alias and confirms any existing local target.
func (desktop *Desktop) localTagDialog(engine domain.Engine, image domain.LocalImage) {
	target := widget.NewEntry()
	target.SetPlaceHolder(desktop.tr("registry.example.com/project/image:v1"))
	desktop.form("Add local image tag", "Check tag", "Cancel", []*widget.FormItem{widget.NewFormItem(desktop.tr("Local image ID"), widget.NewLabel(image.ID)), widget.NewFormItem(desktop.tr("New reference"), target)}, func(confirmed bool) {
		if confirmed {
			desktop.checkLocalTag(engine, image, strings.TrimSpace(target.Text))
		}
	}).Show()
}

// checkLocalTag retains original aliases and obtains consent before replacing another ID.
func (desktop *Desktop) checkLocalTag(engine domain.Engine, image domain.LocalImage, target string) {
	desktop.run("Checking local target tag...", func(ctx context.Context) (any, error) { return desktop.presenter.Local.TagID(ctx, engine, target) }, func(result any) {
		previous := result.(string)
		apply := func() {
			desktop.run("Adding local image tag...", func(ctx context.Context) (any, error) {
				return nil, desktop.presenter.Local.Tag(ctx, domain.LocalSource{Engine: engine, ImageID: image.ID}, target, previous)
			}, func(any) { desktop.closeDrawer(); desktop.localImages.refresh() })
		}
		if previous != "" && previous != image.ID {
			desktop.confirm("Replace local tag?", desktop.tr("The local tag points to another image. Replace this alias? The original image and its other tags are retained.")+"\n"+target+"\n"+previous, func(confirmed bool) {
				if confirmed {
					apply()
				}
			})
			return
		}
		apply()
	})
}
