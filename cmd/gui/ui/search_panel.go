package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"filesearch/internal/search"
	"strings"
)

// SearchPanel contains all search-related widgets
type SearchPanel struct {
	PatternEntry    *widget.Entry
	ExtensionEntry  *widget.Entry
	IgnoreCaseCheck *widget.Check
	DirsLabel       *widget.Label
	SelectedDirs    []string
	addDirBtn       *widget.Button
	clearDirsBtn    *widget.Button
	searchBtn       *widget.Button
	stopBtn         *widget.Button
}

// CreateSearchPanel creates and returns search panel widgets
func CreateSearchPanel(window fyne.Window) *SearchPanel {
	panel := &SearchPanel{
		PatternEntry: widget.NewEntry(),
		ExtensionEntry: widget.NewEntry(),
		IgnoreCaseCheck: widget.NewCheck("Ignore case", nil),
		DirsLabel: widget.NewLabel(""),
		SelectedDirs: make([]string, 0),
	}
	
	panel.PatternEntry.SetPlaceHolder("File name")
	panel.ExtensionEntry.SetPlaceHolder("txt, doc")
	panel.DirsLabel.Wrapping = fyne.TextWrapWord
	
	panel.addDirBtn = widget.NewButton("Add Search Directory", func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				search.LogError("Failed to open directory: %v", err)
				dialog.ShowError(err, window)
				return
			}
			if uri == nil {
				search.LogWarning("No directory selected")
				return
			}
			panel.SelectedDirs = append(panel.SelectedDirs, uri.Path())
			search.LogInfo("Added directory: %s", uri.Path())
			panel.updateDirsLabel()
		}, window)
		d.Resize(fyne.NewSize(500, 400))
		d.Show()
	})
	
	panel.clearDirsBtn = widget.NewButton("Clear Search Directories", func() {
		panel.SelectedDirs = make([]string, 0)
		panel.updateDirsLabel()
	})
	
	// Update initial label
	panel.updateDirsLabel()
	
	return panel
}

// AddSearchButton adds the search button to the panel
func (p *SearchPanel) AddSearchButton(btn *widget.Button) {
	p.searchBtn = btn
}

// AddStopButton adds the stop button to the panel
func (p *SearchPanel) AddStopButton(btn *widget.Button) {
	p.stopBtn = btn
}

// GetContent returns the container with all search panel widgets
func (p *SearchPanel) GetContent() *fyne.Container {
	return container.NewVBox(
		p.PatternEntry,
		p.ExtensionEntry,
		p.IgnoreCaseCheck,
		p.searchBtn,
		p.stopBtn,
		p.addDirBtn,
		p.clearDirsBtn,
		p.DirsLabel,
	)
}

func (p *SearchPanel) updateDirsLabel() {
	if len(p.SelectedDirs) == 0 {
		p.DirsLabel.SetText("No directories selected\n(will search everywhere)")
	} else {
		p.DirsLabel.SetText("Selected directories:\n" + strings.Join(p.SelectedDirs, "\n"))
	}
} 