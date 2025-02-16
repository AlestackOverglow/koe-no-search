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
	
	switch runtime.GOOS {
	case "windows":
		// Use explorer.exe directly
		explorerPath := "explorer.exe"
		if windir := os.Getenv("WINDIR"); windir != "" {
			explorerPath = filepath.Join(windir, "explorer.exe")
		}
		
		// Get absolute path to the file
		absPath, err := filepath.Abs(path)
		if err != nil {
			search.LogError("Failed to get absolute path: %v", err)
			dialog.ShowError(fmt.Errorf("Failed to get file path: %v", err), nil)
			return
		}
		
		// Launch explorer with /select parameter
		cmd := exec.Command(explorerPath, "/select,", absPath)
		if err := cmd.Run(); err != nil {
			search.LogError("Failed to open in explorer: %v", err)
			dialog.ShowError(fmt.Errorf("Failed to open in explorer: %v", err), nil)
		}
		
	case "darwin":
		cmd := exec.Command("open", "-R", path)
		if err := cmd.Run(); err != nil {
			search.LogError("Failed to open in Finder: %v", err)
			dialog.ShowError(fmt.Errorf("Failed to open in Finder: %v", err), nil)
		}
		
	default: // Linux and other Unix-like systems
		cmd := exec.Command("xdg-open", filepath.Dir(path))
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
	patternEntry.SetPlaceHolder("Search pattern")
	
	extensionEntry := widget.NewEntry()
	extensionEntry.SetPlaceHolder("File extension (e.g., .txt)")
	
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
	
	// Create search button
	searchBtn := widget.NewButton("Start Search", nil)
	
	// Channel for stop signal
	var stopChan chan struct{}
	
	searchBtn.OnTapped = func() {
		// Create new stop channel
		stopChan = make(chan struct{})
		
		// If no directories selected, use all available drives
		searchDirs := selectedDirs
		if len(searchDirs) == 0 {
			searchDirs = getAllDrives()
			search.LogInfo("No directories selected, using all drives: %v", searchDirs)
		}
		
		// Disable search button and show progress
		searchBtn.Disable()
		progress.Show()
		
		opts := search.SearchOptions{
			RootDirs:   searchDirs,
			Pattern:    patternEntry.Text,
			Extension:  extensionEntry.Text,
			MaxWorkers: int(workersSlider.Value),
			IgnoreCase: ignoreCaseCheck.Checked,
			BufferSize: int(bufferSlider.Value),
			StopChan:   stopChan,
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
						progress.Hide()
						search.LogInfo("Search completed. Found %d files, %d errors", count, errors)
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
					
				case <-stopChan:
					// Stop signal received
					if needsUpdate {
						resultsList.Refresh()
					}
					search.LogInfo("Search stopped by user. Found %d files, %d errors", count, errors)
					searchBtn.Enable()
					progress.Hide()
					return
				}
			}
		}()
	}
	
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
		searchBtn,
		stopBtn,
		progress,
		exitBtn,
		widget.NewSeparator(), // Separator before About section
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
	
	w.ShowAndRun()
} 