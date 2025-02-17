package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"filesearch/cmd/gui/explorer"
)

// FileListItem represents an item in the file list
type FileListItem struct {
	Path string
	Size int64
}

// CreateResultsList creates and returns a list widget for search results
func CreateResultsList() (*widget.List, *[]FileListItem) {
	var foundFiles []FileListItem
	
	resultsList := widget.NewList(
		func() int {
			return len(foundFiles)
		},
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewLabel("Template Text That Is Long Enough"),
				widget.NewSeparator(),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			container := o.(*fyne.Container)
			label := container.Objects[0].(*widget.Label)
			
			file := foundFiles[i]
			
			// Format file size
			var sizeStr string
			switch {
			case file.Size >= 1024*1024*1024:
				sizeStr = fmt.Sprintf("%.2f GB", float64(file.Size)/(1024*1024*1024))
			case file.Size >= 1024*1024:
				sizeStr = fmt.Sprintf("%.2f MB", float64(file.Size)/(1024*1024))
			case file.Size >= 1024:
				sizeStr = fmt.Sprintf("%.2f KB", float64(file.Size)/1024)
			default:
				sizeStr = fmt.Sprintf("%d B", file.Size)
			}
			
			label.SetText(fmt.Sprintf("%s (%s)", file.Path, sizeStr))
		},
	)
	
	// Add double-click handler
	resultsList.OnSelected = func(id widget.ListItemID) {
		if id < len(foundFiles) {
			go explorer.ShowInExplorer(foundFiles[id].Path)
		}
		resultsList.UnselectAll()
	}
	
	return resultsList, &foundFiles
} 