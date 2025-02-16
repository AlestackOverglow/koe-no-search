package main

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	
	"filesearch/internal/search"
)

// FileListItem represents an item in the file list
type FileListItem struct {
	Path string
	Size int64
}

// ShowInExplorer opens the file location in explorer
func ShowInExplorer(path string) {
	path = filepath.Clean(path)
	
	// Validate path
	if _, err := os.Stat(path); err != nil {
		search.LogError("File no longer exists or inaccessible: %v", err)
		dialog.ShowError(fmt.Errorf("File no longer exists or inaccessible: %v", err), nil)
		return
	}

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		search.LogError("Failed to get absolute path: %v", err)
		dialog.ShowError(fmt.Errorf("Failed to get file path: %v", err), nil)
		return
	}

	// Check if path is accessible
	if _, err := os.Stat(absPath); err != nil {
		search.LogError("Path is not accessible: %v", err)
		dialog.ShowError(fmt.Errorf("Path is not accessible: %v", err), nil)
		return
	}

	switch runtime.GOOS {
	case "windows":
		// Use explorer.exe directly
		explorerPath := "explorer.exe"
		if windir := os.Getenv("WINDIR"); windir != "" {
			explorerPath = filepath.Join(windir, "explorer.exe")
			if _, err := os.Stat(explorerPath); err != nil {
				explorerPath = "explorer.exe" // Fallback to PATH
			}
		}

		// Try to open file with selection first
		cmd := exec.Command(explorerPath, "/select,", absPath)
		if err := cmd.Run(); err != nil {
			search.LogWarning("Failed to open file with selection, trying to open directory: %v", err)
			
			// If selection fails, try to open the directory
			dirPath := filepath.Dir(absPath)
			cmd = exec.Command(explorerPath, dirPath)
			if err := cmd.Run(); err != nil {
				search.LogError("Failed to open directory in explorer: %v", err)
				dialog.ShowError(fmt.Errorf("Failed to open in explorer: %v", err), nil)
			}
		}
		
	case "darwin":
		cmd := exec.Command("open", "-R", absPath)
		if err := cmd.Run(); err != nil {
			search.LogError("Failed to open in Finder: %v", err)
			dialog.ShowError(fmt.Errorf("Failed to open in Finder: %v", err), nil)
		}
		
	default: // Linux and other Unix-like systems
		dirPath := filepath.Dir(absPath)
		cmd := exec.Command("xdg-open", dirPath)
		if err := cmd.Run(); err != nil {
			search.LogError("Failed to open in file manager: %v", err)
			dialog.ShowError(fmt.Errorf("Failed to open in file manager: %v", err), nil)
		}
	}
}

// getAllDrives returns a list of all available drives in Windows/Linux
func getAllDrives() []string {
	if runtime.GOOS == "windows" {
		drives := []string{}
		for _, drive := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
			drivePath := string(drive) + ":\\"
			_, err := os.Stat(drivePath)
			if err == nil {
				drives = append(drives, drivePath)
			}
		}
		return drives
	} else {
		// For Linux return root directory
		return []string{"/"}
	}
}

func main() {
	// Initialize logger
	search.InitLogger()
	defer search.CloseLogger()
	
	a := app.New()
	w := a.NewWindow("Koe no Search")
	
	// Create input fields
	patternEntry := widget.NewEntry()
	patternEntry.SetPlaceHolder("Search patterns (comma-separated, e.g.: *.txt, *.doc)")
	
	extensionEntry := widget.NewEntry()
	extensionEntry.SetPlaceHolder("File extensions (comma-separated, e.g.: txt, doc)")
	
	// Helper function to split comma-separated values
	splitCommaList := func(s string) []string {
		if s == "" {
			return nil
		}
		parts := strings.Split(s, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
		return result
	}
	
	ignoreCaseCheck := widget.NewCheck("Ignore case", nil)
	
	workersSlider := widget.NewSlider(1, float64(runtime.NumCPU()*2))
	workersSlider.SetValue(float64(runtime.NumCPU()))
	
	workersLabel := widget.NewLabel(fmt.Sprintf("Workers: %d", runtime.NumCPU()))
	workersSlider.OnChanged = func(v float64) {
		workersLabel.SetText(fmt.Sprintf("Workers: %d", int(v)))
	}
	
	bufferSlider := widget.NewSlider(100, 10000)
	bufferSlider.SetValue(1000)
	
	bufferLabel := widget.NewLabel("Buffer size: 1000")
	bufferSlider.OnChanged = func(v float64) {
		bufferLabel.SetText(fmt.Sprintf("Buffer size: %d", int(v)))
	}
	
	// Create progress bar
	progress := widget.NewProgressBarInfinite()
	progress.Hide()
	
	// Create list for selected directories
	selectedDirs := make([]string, 0)
	dirsLabel := widget.NewLabel("")
	dirsLabel.Wrapping = fyne.TextWrapWord
	updateDirsLabel := func() {
		if len(selectedDirs) == 0 {
			dirsLabel.SetText("No directories selected\n(will search everywhere)")
		} else {
			dirsLabel.SetText("Selected directories:\n" + strings.Join(selectedDirs, "\n"))
		}
	}
	updateDirsLabel()
	
	// Create list for results
	var foundFiles []FileListItem
	resultsList := widget.NewList(
		func() int {
			return len(foundFiles)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template Text That Is Long Enough")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			file := foundFiles[i]
			label := o.(*widget.Label)
			
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
			go ShowInExplorer(foundFiles[id].Path)
		}
		resultsList.UnselectAll()
	}
	
	// Create directory selection button
	addDirBtn := widget.NewButton("Add Directory", func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				search.LogError("Failed to open directory: %v", err)
				dialog.ShowError(err, w)
				return
			}
			if uri == nil {
				search.LogWarning("No directory selected")
				return
			}
			selectedDirs = append(selectedDirs, uri.Path())
			search.LogInfo("Added directory: %s", uri.Path())
			updateDirsLabel()
		}, w)
		d.Resize(fyne.NewSize(500, 400))
		d.Show()
	})
	
	// Create clear directories button
	clearDirsBtn := widget.NewButton("Clear Directories", func() {
		selectedDirs = make([]string, 0)
		updateDirsLabel()
	})
	
	// File operations frame
	fileOpFrame := widget.NewCard("File Operations", "", nil)
	
	// Operation type selection
	opTypeSelect := widget.NewSelect([]string{
		"No Operation",
		"Copy Files",
		"Move Files",
		"Delete Files",
	}, nil)
	opTypeSelect.SetSelected("No Operation")

	// Target directory selection
	var targetDir string
	targetDirLabel := widget.NewLabel("Target Directory: Not selected")
	targetDirLabel.Wrapping = fyne.TextWrapWord

	selectTargetBtn := widget.NewButton("Select Target Directory", func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if uri != nil {
				targetDir = uri.Path()
				targetDirLabel.SetText("Target Directory: " + targetDir)
			}
		}, w)
		d.Resize(fyne.NewSize(500, 400))
		d.Show()
	})

	// Conflict resolution policy
	conflictPolicy := widget.NewSelect([]string{
		"Skip",
		"Overwrite",
		"Rename",
	}, nil)
	conflictPolicy.SetSelected("Skip")

	// Create file operations container
	fileOpContent := container.NewVBox(
		widget.NewLabel("Operation:"),
		opTypeSelect,
		widget.NewSeparator(),
		targetDirLabel,
		selectTargetBtn,
		widget.NewSeparator(),
		widget.NewLabel("On File Conflict:"),
		conflictPolicy,
	)
	fileOpFrame.SetContent(fileOpContent)
	
	// Create search button
	searchBtn := widget.NewButton("Start Search", nil)
	
	// Add label for search time
	searchTimeLabel := widget.NewLabel("")
	
	// Channel for stop signal
	var stopChan chan struct{}
	
	// Create file operations button
	fileOpBtn := widget.NewButton("Apply Operation", func() {
		if len(foundFiles) == 0 {
			dialog.ShowError(fmt.Errorf("No files found to process"), w)
			return
		}

		// Validate file operations
		var fileOp search.FileOperationOptions
		switch opTypeSelect.Selected {
		case "Copy Files":
			fileOp.Operation = search.CopyFiles
		case "Move Files":
			fileOp.Operation = search.MoveFiles
		case "Delete Files":
			fileOp.Operation = search.DeleteFiles
		default:
			dialog.ShowError(fmt.Errorf("Please select an operation"), w)
			return
		}

		if fileOp.Operation != search.NoOperation && fileOp.Operation != search.DeleteFiles {
			if targetDir == "" {
				dialog.ShowError(fmt.Errorf("Please select target directory"), w)
				return
			}
		}

		fileOp.TargetDir = targetDir

		switch conflictPolicy.Selected {
		case "Skip":
			fileOp.ConflictPolicy = search.Skip
		case "Overwrite":
			fileOp.ConflictPolicy = search.Overwrite
		case "Rename":
			fileOp.ConflictPolicy = search.Rename
		}

		// Create progress dialog
		progress := dialog.NewProgress("Processing Files", "Processing files...", w)
		progress.Show()

		// Process files in a goroutine
		go func() {
			processor := search.NewFileOperationProcessor(runtime.NumCPU())
			processor.Start()
			defer processor.Stop()

			total := len(foundFiles)
			for i, file := range foundFiles {
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
				fmt.Sprintf("Processed %d files", total), w)

			// Clear results if files were moved or deleted
			if fileOp.Operation == search.MoveFiles || fileOp.Operation == search.DeleteFiles {
				foundFiles = make([]FileListItem, 0)
				resultsList.Refresh()
			}
		}()
	})
	fileOpBtn.Disable() // Disabled by default

	// Create stop button
	stopBtn := widget.NewButton("Stop Search", func() {
		if stopChan != nil {
			search.LogInfo("Search stop requested by user")
			close(stopChan)
			stopChan = nil
		}
	})
	
	// Create exit button
	exitBtn := widget.NewButton("Exit", func() {
		search.LogInfo("Application exit requested by user")
		if stopChan != nil {
			close(stopChan)
		}
		a.Quit()
	})
	
	// Create about section
	authorLabel := widget.NewLabel("Author: AlestackOverglow")
	repoURL, err := url.Parse("https://github.com/AlestackOverglow/koe-no-search")
	if err != nil {
		search.LogError("Failed to parse repository URL: %v", err)
	}
	repoLink := widget.NewHyperlink("GitHub Repository", repoURL)
	repoLink.OnTapped = func() {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmdPath := os.Getenv("COMSPEC")
			if cmdPath == "" {
				cmdPath = `C:\Windows\System32\cmd.exe`
			}
			cmd = exec.Command(cmdPath, "/c", "start", repoURL.String())
		case "darwin":
			cmd = exec.Command("open", repoURL.String())
		default:
			cmd = exec.Command("xdg-open", repoURL.String())
		}
		
		if err := cmd.Run(); err != nil {
			search.LogError("Failed to open URL: %v", err)
			dialog.ShowError(fmt.Errorf("Failed to open URL: %v", err), w)
		}
	}
	aboutBox := container.NewVBox(
		authorLabel,
		repoLink,
	)
	
	// Layout
	dirButtons := container.NewHBox(addDirBtn, clearDirsBtn)
	
	inputs := container.NewVBox(
		patternEntry,
		extensionEntry,
		ignoreCaseCheck,
		workersLabel,
		workersSlider,
		bufferLabel,
		bufferSlider,
		dirButtons,
		dirsLabel,
		widget.NewSeparator(),
		fileOpFrame,
		widget.NewSeparator(),
		searchBtn,
		searchTimeLabel,
		fileOpBtn,
		stopBtn,
		progress,
		exitBtn,
		widget.NewSeparator(),
		aboutBox,
	)
	
	content := container.NewHSplit(
		inputs,
		container.NewScroll(resultsList),
	)
	
	w.SetContent(content)
	w.Resize(fyne.NewSize(800, 600))
	
	// Set window close handler
	w.SetCloseIntercept(func() {
		a.Quit()
	})
	
	searchBtn.OnTapped = func() {
		// Create new stop channel
		stopChan = make(chan struct{})
		
		// If no directories selected, use all available drives
		searchDirs := selectedDirs
		if len(searchDirs) == 0 {
			searchDirs = getAllDrives()
			search.LogInfo("No directories selected, using all drives: %v", searchDirs)
		}
		
		// Disable buttons and show progress
		searchBtn.Disable()
		fileOpBtn.Disable()
		progress.Show()
		
		// Reset search time label
		searchTimeLabel.SetText("Searching...")
		
		// Record start time
		startTime := time.Now()
		
		opts := search.SearchOptions{
			RootDirs:    searchDirs,
			Patterns:    splitCommaList(patternEntry.Text),
			Extensions:  splitCommaList(extensionEntry.Text),
			MaxWorkers:  int(workersSlider.Value),
			IgnoreCase:  ignoreCaseCheck.Checked,
			BufferSize:  int(bufferSlider.Value),
			StopChan:    stopChan,
			ExcludeDirs: []string{},
		}

		// If target directory is set, add it to excluded directories
		if targetDir != "" {
			opts.ExcludeDirs = append(opts.ExcludeDirs, targetDir)
		}
		
		search.LogInfo("Starting search with options: %+v", opts)
		
		// Clear previous results
		foundFiles = make([]FileListItem, 0)
		resultsList.Refresh()
		
		// Create channel for results
		count := 0
		errors := 0
		
		// Process results in a goroutine with panic recovery
		go func() {
			defer func() {
				if r := recover(); r != nil {
					search.LogError("Panic during search: %v", r)
					// Restore interface in case of panic
					searchBtn.Enable()
					fileOpBtn.Disable()
					progress.Hide()
					dialog.ShowError(fmt.Errorf("Search error: %v", r), w)
					if stopChan != nil {
						close(stopChan)
					}
				}
			}()
			
			// Create channel for results
			results := search.Search(opts)
			
			// Channel for UI updates
			updateTicker := time.NewTicker(100 * time.Millisecond)
			defer updateTicker.Stop()
			
			needsUpdate := false
			
			// Process results
			for {
				select {
				case result, ok := <-results:
					if !ok {
						// Channel closed, finish search
						if needsUpdate {
							resultsList.Refresh()
						}
						searchBtn.Enable()
						if len(foundFiles) > 0 {
							fileOpBtn.Enable()
						}
						progress.Hide()
						
						// Calculate and display search time
						duration := time.Since(startTime)
						seconds := int(duration.Seconds())
						milliseconds := int(duration.Milliseconds()) % 1000
						searchTimeLabel.SetText(fmt.Sprintf("Search completed in %d.%03d seconds\n(%d files found, %d errors)",
							seconds,
							milliseconds,
							count,
							errors))
						
						search.LogInfo("Search completed in %v. Found %d files, %d errors", duration, count, errors)
						return
					}
					count++
					if result.Error != nil {
						errors++
						search.LogError("Error processing file %s: %v", result.Path, result.Error)
					} else {
						search.LogDebug("Found file: %s (Size: %d bytes)", result.Path, result.Size)
						foundFiles = append(foundFiles, FileListItem{
							Path: result.Path,
							Size: result.Size,
						})
						needsUpdate = true
					}
					
				case <-updateTicker.C:
					// Update UI only if there are new results
					if needsUpdate {
						resultsList.Refresh()
						needsUpdate = false
					}
					// Update search time
					duration := time.Since(startTime)
					seconds := int(duration.Seconds())
					milliseconds := int(duration.Milliseconds()) % 1000
					searchTimeLabel.SetText(fmt.Sprintf("Searching... %d.%03d seconds",
						seconds,
						milliseconds))
					
				case <-stopChan:
					// Stop signal received
					if needsUpdate {
						resultsList.Refresh()
					}
					searchBtn.Enable()
					if len(foundFiles) > 0 {
						fileOpBtn.Enable()
					}
					progress.Hide()
					
					// Calculate and display final search time
					duration := time.Since(startTime)
					seconds := int(duration.Seconds())
					milliseconds := int(duration.Milliseconds()) % 1000
					searchTimeLabel.SetText(fmt.Sprintf("Search stopped after %d.%03d seconds\n(%d files found, %d errors)",
						seconds,
						milliseconds,
						count,
						errors))
					
					search.LogInfo("Search stopped by user after %v. Found %d files, %d errors", duration, count, errors)
					return
				}
			}
		}()
	}
	
	w.ShowAndRun()
} 