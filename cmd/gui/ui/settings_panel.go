package ui

import (
	"fmt"
	"runtime"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// SettingsPanel contains all settings-related widgets
type SettingsPanel struct {
	WorkersSlider        *widget.Slider
	WorkersLabel         *widget.Label
	BufferSlider        *widget.Slider
	BufferLabel         *widget.Label
	MinSizeEntry        *widget.Entry
	MaxSizeEntry        *widget.Entry
	MinAgeEntry         *widget.Entry
	MaxAgeEntry         *widget.Entry
	ExcludeHiddenCheck  *widget.Check
	FollowSymlinksCheck *widget.Check
	DeduplicateCheck    *widget.Check
	UseMMapCheck        *widget.Check
	MinMMapSizeEntry    *widget.Entry
	PriorityPanel       *PriorityDirsPanel
}

// CreateSettingsPanel creates and returns settings panel widgets
func CreateSettingsPanel(window fyne.Window) *SettingsPanel {
	panel := &SettingsPanel{
		WorkersSlider: widget.NewSlider(1, float64(runtime.NumCPU()*2)),
		WorkersLabel: widget.NewLabel(fmt.Sprintf("Workers: %d", runtime.NumCPU())),
		BufferSlider: widget.NewSlider(100, 10000),
		BufferLabel: widget.NewLabel("Buffer size: 1000"),
		MinSizeEntry: widget.NewEntry(),
		MaxSizeEntry: widget.NewEntry(),
		MinAgeEntry: widget.NewEntry(),
		MaxAgeEntry: widget.NewEntry(),
		ExcludeHiddenCheck: widget.NewCheck("Exclude hidden files", nil),
		FollowSymlinksCheck: widget.NewCheck("Follow symbolic links", nil),
		DeduplicateCheck: widget.NewCheck("Remove duplicates", nil),
		UseMMapCheck: widget.NewCheck("Use memory mapping", nil),
		MinMMapSizeEntry: widget.NewEntry(),
		PriorityPanel: CreatePriorityDirsPanel(window),
	}
	
	// Set initial values
	panel.WorkersSlider.SetValue(float64(runtime.NumCPU()))
	panel.BufferSlider.SetValue(1000)
	
	// Set placeholders
	panel.MinSizeEntry.SetPlaceHolder("1KB, 1.5MB, 2GB")
	panel.MaxSizeEntry.SetPlaceHolder("1KB, 1.5MB, 2GB")
	panel.MinAgeEntry.SetPlaceHolder("1h, 2d, 1w, 1m")
	panel.MaxAgeEntry.SetPlaceHolder("1h, 2d, 1w, 1m")
	panel.MinMMapSizeEntry.SetPlaceHolder("1MB")
	
	// Set up callbacks
	panel.WorkersSlider.OnChanged = func(v float64) {
		panel.WorkersLabel.SetText(fmt.Sprintf("Workers: %d", int(v)))
	}
	
	panel.BufferSlider.OnChanged = func(v float64) {
		panel.BufferLabel.SetText(fmt.Sprintf("Buffer size: %d", int(v)))
	}
	
	panel.UseMMapCheck.OnChanged = func(checked bool) {
		if checked {
			panel.MinMMapSizeEntry.Enable()
		} else {
			panel.MinMMapSizeEntry.Disable()
		}
	}
	
	// Initially disable mmap size entry
	panel.MinMMapSizeEntry.Disable()
	
	return panel
}

// GetContent returns the container with all settings panel widgets
func (p *SettingsPanel) GetContent() *fyne.Container {
	return container.NewVBox(
		widget.NewLabel("Performance:"),
		p.WorkersLabel,
		p.WorkersSlider,
		widget.NewSeparator(),
		p.BufferLabel,
		p.BufferSlider,
		widget.NewSeparator(),
		widget.NewLabel("Filters:"),
		widget.NewLabel("Min file size:"),
		p.MinSizeEntry,
		widget.NewLabel("Max file size:"),
		p.MaxSizeEntry,
		widget.NewLabel("Min file age:"),
		p.MinAgeEntry,
		widget.NewLabel("Max file age:"),
		p.MaxAgeEntry,
		widget.NewSeparator(),
		widget.NewLabel("Processing:"),
		p.ExcludeHiddenCheck,
		p.FollowSymlinksCheck,
		p.DeduplicateCheck,
		widget.NewSeparator(),
		widget.NewLabel("Memory Mapping:"),
		p.UseMMapCheck,
		widget.NewLabel("For files > 1MB"),
		p.MinMMapSizeEntry,
		p.PriorityPanel.GetContent(),
	)
} 