package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// SettingsPanel contains all settings-related widgets
type SettingsPanel struct {
	MinSizeEntry        *widget.Entry
	MaxSizeEntry        *widget.Entry
	MinAgeEntry         *widget.Entry
	MaxAgeEntry         *widget.Entry
	ExcludeHiddenCheck  *widget.Check
	FollowSymlinksCheck *widget.Check
	DeduplicateCheck    *widget.Check
	UseMMapCheck        *widget.Check
}

// CreateSettingsPanel creates and returns settings panel widgets
func CreateSettingsPanel(window fyne.Window) *SettingsPanel {
	panel := &SettingsPanel{
		MinSizeEntry: widget.NewEntry(),
		MaxSizeEntry: widget.NewEntry(),
		MinAgeEntry: widget.NewEntry(),
		MaxAgeEntry: widget.NewEntry(),
		ExcludeHiddenCheck: widget.NewCheck("Exclude hidden files", nil),
		FollowSymlinksCheck: widget.NewCheck("Follow symbolic links", nil),
		DeduplicateCheck: widget.NewCheck("Remove duplicates", nil),
		UseMMapCheck: widget.NewCheck("Use memory mapping", nil),
	}
	
	// Set default values
	panel.UseMMapCheck.SetChecked(true)
	
	// Set placeholders
	panel.MinSizeEntry.SetPlaceHolder("1KB, 1.5MB, 2GB")
	panel.MaxSizeEntry.SetPlaceHolder("1KB, 1.5MB, 2GB")
	panel.MinAgeEntry.SetPlaceHolder("1h, 2d, 1w, 1m")
	panel.MaxAgeEntry.SetPlaceHolder("1h, 2d, 1w, 1m")
	
	return panel
}

// GetContent returns the container with all settings panel widgets
func (p *SettingsPanel) GetContent() *fyne.Container {
	return container.NewVBox(
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
	)
} 