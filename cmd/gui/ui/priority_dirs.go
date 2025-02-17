package ui

import (
	"strings"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// PriorityDirsPanel contains all priority directories related widgets
type PriorityDirsPanel struct {
	PriorityDirs     []string
	LowPriorityDirs  []string
	ExcludedDirs     []string
	PriorityLabel    *widget.Label
	LowPriorityLabel *widget.Label
	ExcludedLabel    *widget.Label
	AddPriorityBtn   *widget.Button
	AddLowPriorityBtn *widget.Button
	AddExcludedBtn   *widget.Button
	ClearAllBtn      *widget.Button
}

// CreatePriorityDirsPanel creates and returns priority directories panel widgets
func CreatePriorityDirsPanel(window fyne.Window) *PriorityDirsPanel {
	panel := &PriorityDirsPanel{
		PriorityDirs: make([]string, 0),
		LowPriorityDirs: make([]string, 0),
		ExcludedDirs: make([]string, 0),
		PriorityLabel: widget.NewLabel(""),
		LowPriorityLabel: widget.NewLabel(""),
		ExcludedLabel: widget.NewLabel(""),
	}
	
	panel.PriorityLabel.Wrapping = fyne.TextWrapWord
	panel.LowPriorityLabel.Wrapping = fyne.TextWrapWord
	panel.ExcludedLabel.Wrapping = fyne.TextWrapWord
	
	panel.AddPriorityBtn = widget.NewButton("Add priority", func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			panel.PriorityDirs = append(panel.PriorityDirs, uri.Path())
			panel.updateLabels()
		}, window)
		d.Show()
	})
	
	panel.AddLowPriorityBtn = widget.NewButton("Add low priority", func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			panel.LowPriorityDirs = append(panel.LowPriorityDirs, uri.Path())
			panel.updateLabels()
		}, window)
		d.Show()
	})
	
	panel.AddExcludedBtn = widget.NewButton("Add excluded", func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			panel.ExcludedDirs = append(panel.ExcludedDirs, uri.Path())
			panel.updateLabels()
		}, window)
		d.Show()
	})
	
	panel.ClearAllBtn = widget.NewButton("Clear priorities", func() {
		panel.PriorityDirs = nil
		panel.LowPriorityDirs = nil
		panel.ExcludedDirs = nil
		panel.updateLabels()
	})
	
	return panel
}

// GetContent returns the container with all priority directories panel widgets
func (p *PriorityDirsPanel) GetContent() *fyne.Container {
	buttons := container.NewVBox(
		widget.NewLabel("Priority Directories:"),
		p.AddPriorityBtn,
		p.AddLowPriorityBtn,
		p.AddExcludedBtn,
		p.ClearAllBtn,
	)

	return buttons
}

func (p *PriorityDirsPanel) updateLabels() {
	if len(p.PriorityDirs) > 0 {
		p.PriorityLabel.SetText(strings.Join(p.PriorityDirs, "\n"))
	} else {
		p.PriorityLabel.SetText("")
	}
	if len(p.LowPriorityDirs) > 0 {
		p.LowPriorityLabel.SetText(strings.Join(p.LowPriorityDirs, "\n"))
	} else {
		p.LowPriorityLabel.SetText("")
	}
	if len(p.ExcludedDirs) > 0 {
		p.ExcludedLabel.SetText(strings.Join(p.ExcludedDirs, "\n"))
	} else {
		p.ExcludedLabel.SetText("")
	}
} 