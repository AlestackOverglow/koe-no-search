package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// SettingsPanel represents the settings panel
type SettingsPanel struct {
	IgnoreCaseCheck *widget.Check
}

// CreateSettingsPanel creates a new settings panel
func CreateSettingsPanel(window fyne.Window) *SettingsPanel {
	panel := &SettingsPanel{
		IgnoreCaseCheck: widget.NewCheck("Ignore Case", nil),
	}
	
	// Set default values
	panel.IgnoreCaseCheck.SetChecked(true)
	
	return panel
}

// GetContent returns the panel content
func (p *SettingsPanel) GetContent() fyne.CanvasObject {
	return container.NewVBox(
		p.IgnoreCaseCheck,
	)
} 