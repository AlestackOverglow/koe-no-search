package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"filesearch/internal/search"
	"runtime"
)

// FileOperationsPanel contains all file operation related widgets
type FileOperationsPanel struct {
	OpTypeSelect    *widget.Select
	TargetDirLabel  *widget.Label
	SelectTargetBtn *widget.Button
	ConflictPolicy  *widget.Select
	TargetDir       string
	OperationBtn    *widget.Button
}

// CreateFileOperationsPanel creates and returns file operations panel widgets
func CreateFileOperationsPanel(window fyne.Window, foundFiles *[]FileListItem) *FileOperationsPanel {
	panel := &FileOperationsPanel{
		OpTypeSelect: widget.NewSelect([]string{
			"No Operation",
			"Copy Files",
			"Move Files",
			"Delete Files",
		}, nil),
		TargetDirLabel: widget.NewLabel("Target Directory: "),
		ConflictPolicy: widget.NewSelect([]string{
			"Skip",
			"Overwrite",
			"Rename",
		}, nil),
		OperationBtn: widget.NewButton("Apply Operation", nil),
	}
	
	panel.TargetDirLabel.Wrapping = fyne.TextWrapWord
	panel.OpTypeSelect.SetSelected("No Operation")
	panel.ConflictPolicy.SetSelected("Skip")
	
	panel.SelectTargetBtn = widget.NewButton("Select Target Directory", func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if uri != nil {
				panel.TargetDir = uri.Path()
				panel.TargetDirLabel.SetText("Target Directory: " + panel.TargetDir)
			}
		}, window)
		d.Resize(fyne.NewSize(500, 400))
		d.Show()
	})
	
	panel.OperationBtn.OnTapped = func() {
		if len(*foundFiles) == 0 {
			dialog.ShowError(fmt.Errorf("No files found to process"), window)
			return
		}

		// Validate file operations
		var fileOp search.FileOperationOptions
		switch panel.OpTypeSelect.Selected {
		case "Copy Files":
			fileOp.Operation = search.CopyFiles
		case "Move Files":
			fileOp.Operation = search.MoveFiles
		case "Delete Files":
			fileOp.Operation = search.DeleteFiles
		default:
			dialog.ShowError(fmt.Errorf("Please select an operation"), window)
			return
		}

		if fileOp.Operation != search.NoOperation && fileOp.Operation != search.DeleteFiles {
			if panel.TargetDir == "" {
				dialog.ShowError(fmt.Errorf("Please select target directory"), window)
				return
			}
		}

		fileOp.TargetDir = panel.TargetDir

		switch panel.ConflictPolicy.Selected {
		case "Skip":
			fileOp.ConflictPolicy = search.Skip
		case "Overwrite":
			fileOp.ConflictPolicy = search.Overwrite
		case "Rename":
			fileOp.ConflictPolicy = search.Rename
		}

		// Create progress dialog
		progress := dialog.NewProgress("Processing Files", "Processing files...", window)
		progress.Show()

		// Process files in a goroutine
		go func() {
			processor := search.NewFileOperationProcessor(runtime.NumCPU())
			processor.Start()
			defer processor.Stop()

			total := len(*foundFiles)
			for i, file := range *foundFiles {
				// Update progress
				progress.SetValue(float64(i) / float64(total))

				// Process file
				if err := search.HandleFileOperation(file.Path, fileOp); err != nil {
					search.LogError("Failed to process file %s: %v", file.Path, err)
					continue
				}
			}

			// Close progress dialog
			progress.Hide()

			// Show completion dialog
			dialog.ShowInformation("Operation Complete", 
				fmt.Sprintf("Processed %d files", total), window)

			// Clear results if files were moved or deleted
			if fileOp.Operation == search.MoveFiles || fileOp.Operation == search.DeleteFiles {
				*foundFiles = make([]FileListItem, 0)
			}
		}()
	}
	
	panel.OperationBtn.Disable() // Disabled by default
	
	return panel
}

// GetContent returns the container with all file operations panel widgets
func (p *FileOperationsPanel) GetContent() *fyne.Container {
	return container.NewVBox(
		widget.NewLabel("Operation:"),
		p.OpTypeSelect,
		widget.NewSeparator(),
		p.TargetDirLabel,
		p.SelectTargetBtn,
		widget.NewSeparator(),
		widget.NewLabel("On File Conflict:"),
		p.ConflictPolicy,
	)
}

// Enable enables the operation button
func (p *FileOperationsPanel) Enable() {
	p.OperationBtn.Enable()
}

// Disable disables the operation button
func (p *FileOperationsPanel) Disable() {
	p.OperationBtn.Disable()
} 