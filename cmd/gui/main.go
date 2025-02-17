package main

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	
	"filesearch/cmd/gui/explorer"
	"filesearch/cmd/gui/ui"
	"filesearch/cmd/gui/utils"
	"filesearch/internal/search"
	"fyne.io/fyne/v2/theme"
	"image/color"
)

// customTheme wraps the default theme to add button transparency
type customTheme struct {
	base fyne.Theme
}

func (t *customTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	c := t.base.Color(n, v)
	if n == theme.ColorNameButton {
		r, g, b, _ := c.RGBA()
		return color.NRGBA{
			R: uint8(r >> 8),
			G: uint8(g >> 8),
			B: uint8(b >> 8),
			A: 180, // Increased transparency (255 is fully opaque)
		}
	} else if n == theme.ColorNamePrimary {
		// Burgundy tint for Start Search button
		return color.NRGBA{
			R: 145,
			G: 85,
			B: 95,
			A: 255,
		}
	}
	return c
}

func (t *customTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(n)
}

func (t *customTheme) Font(s fyne.TextStyle) fyne.Resource {
	return t.base.Font(s)
}

func (t *customTheme) Size(n fyne.ThemeSizeName) float32 {
	return t.base.Size(n)
}

func main() {
	// Initialize logger
	search.InitLogger()
	defer search.CloseLogger()
	
	a := app.New()

	// Set application theme with slightly transparent buttons
	theme := a.Settings().Theme()
	a.Settings().SetTheme(&customTheme{
		base: theme,
	})

	// Set application icon
	if len(search.IconData) > 0 {
		iconResource := fyne.NewStaticResource("icon.png", search.IconData)
		a.SetIcon(iconResource)
	}

	w := a.NewWindow(fmt.Sprintf("Koe no Search v%s", search.Version))
	
	// Create search panel
	searchPanel := ui.CreateSearchPanel(w)
	
	// Create progress bar
	progress := widget.NewProgressBarInfinite()
	progress.Hide()
	
	// Create results list
	resultsList, foundFiles := ui.CreateResultsList()
	
	// Create settings panel
	settingsPanel := ui.CreateSettingsPanel(w)
	
	// Create file operations panel
	fileOpPanel := ui.CreateFileOperationsPanel(w, foundFiles)
	
	// Create search button and add it to search panel
	searchBtn := widget.NewButton("Start Search", nil)
	searchBtn.Importance = widget.HighImportance
	searchPanel.AddSearchButton(searchBtn)
	
	// Add label for search time
	searchTimeLabel := widget.NewLabel("")
	
	// Channel for stop signal
	var stopChan chan struct{}
	
	// Create stop button
	stopBtn := widget.NewButton("Stop Search", func() {
		if stopChan != nil {
			search.LogInfo("Search stop requested by user")
			close(stopChan)
			stopChan = nil
		}
	})
	searchPanel.AddStopButton(stopBtn)
	
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
	
	// Create advanced settings accordion
	advancedSettingsContent := settingsPanel.GetContent()
	advancedSettingsAccordion := widget.NewAccordion(
		widget.NewAccordionItem("Advanced Settings", advancedSettingsContent),
	)
	advancedSettingsAccordion.Close(0) // Closed by default
	
	// Create file operations accordion
	fileOpContent := fileOpPanel.GetContent()
	fileOpAccordion := widget.NewAccordion(
		widget.NewAccordionItem("File Operations", fileOpContent),
	)
	fileOpAccordion.Close(0) // Closed by default
	
	// Layout
	inputs := container.NewVBox(
		searchPanel.GetContent(),
		widget.NewSeparator(),
		advancedSettingsAccordion,
		widget.NewSeparator(),
		fileOpAccordion,
		widget.NewSeparator(),
		searchTimeLabel,
		fileOpPanel.OperationBtn,
		progress,
		widget.NewSeparator(),
		aboutBox,
	)
	
	// Wrap inputs in scroll container with minimum width
	scrollContainer := container.NewVBox(inputs)
	scrolledInputs := container.NewVScroll(scrollContainer)
	minSize := fyne.NewSize(300, 0) // Minimum width 300 pixels
	scrollContainer.Resize(minSize)
	
	// Create split container
	split := container.NewHSplit(
		scrolledInputs,
		container.NewScroll(resultsList),
	)
	split.SetOffset(0.3) // Left part takes 30% of window width
	
	// Set background image if available
	var mainContainer fyne.CanvasObject
	if len(search.BackgroundData) > 0 {
		bgResource := fyne.NewStaticResource("background.png", search.BackgroundData)
		bgImage := canvas.NewImageFromResource(bgResource)
		bgImage.Resize(fyne.NewSize(800, 600))
		bgImage.FillMode = canvas.ImageFillStretch
		bgImage.Translucency = 0.9 // 1.0 - fully transparent, 0.0 - fully opaque
		
		mainContainer = container.NewMax(
			bgImage,
			split,
		)
	} else {
		mainContainer = split
	}
	
	w.SetContent(mainContainer)
	w.Resize(fyne.NewSize(800, 600))
	
	// Set window close handler
	w.SetCloseIntercept(func() {
		a.Quit()
	})
	
	searchBtn.OnTapped = func() {
		// Create new stop channel
		stopChan = make(chan struct{})
		
		// If no directories selected, use all available drives
		searchDirs := searchPanel.SelectedDirs
		if len(searchDirs) == 0 {
			searchDirs = explorer.GetAllDrives()
			search.LogInfo("No directories selected, using all drives: %v", searchDirs)
		}
		
		// Disable buttons and show progress
		searchBtn.Disable()
		fileOpPanel.Disable()
		progress.Show()
		
		// Reset search time label
		searchTimeLabel.SetText("Searching...")
		
		// Record start time
		startTime := time.Now()
		
		opts := search.SearchOptions{
			RootDirs:    searchDirs,
			Patterns:    utils.SplitCommaList(searchPanel.PatternEntry.Text),
			Extensions:  utils.SplitCommaList(searchPanel.ExtensionEntry.Text),
			MaxWorkers:  runtime.NumCPU(),
			IgnoreCase:  searchPanel.IgnoreCaseCheck.Checked,
			BufferSize:  1000,
			StopChan:    stopChan,
		}

		// If target directory is set, add it to excluded directories
		if fileOpPanel.TargetDir != "" {
			opts.ExcludeDirs = append(opts.ExcludeDirs, fileOpPanel.TargetDir)
		}
		
		// Parse and add advanced settings
		if minSize, err := utils.ParseSize(settingsPanel.MinSizeEntry.Text); err == nil && minSize > 0 {
			opts.MinSize = minSize
		}
		if maxSize, err := utils.ParseSize(settingsPanel.MaxSizeEntry.Text); err == nil && maxSize > 0 {
			opts.MaxSize = maxSize
		}
		if minAge, err := utils.ParseAge(settingsPanel.MinAgeEntry.Text); err == nil && minAge > 0 {
			opts.MinAge = minAge
		}
		if maxAge, err := utils.ParseAge(settingsPanel.MaxAgeEntry.Text); err == nil && maxAge > 0 {
			opts.MaxAge = maxAge
		}
		
		opts.ExcludeHidden = settingsPanel.ExcludeHiddenCheck.Checked
		opts.FollowSymlinks = settingsPanel.FollowSymlinksCheck.Checked
		opts.DeduplicateFiles = settingsPanel.DeduplicateCheck.Checked
		opts.UseMMap = settingsPanel.UseMMapCheck.Checked
		
		if opts.UseMMap {
			opts.MinMMapSize = 1024 * 1024 // 1MB default
		}
		
		search.LogInfo("Starting search with options: %+v", opts)
		
		// Clear previous results
		*foundFiles = make([]ui.FileListItem, 0)
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
					fileOpPanel.Disable()
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
						if len(*foundFiles) > 0 {
							fileOpPanel.Enable()
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
						
						*foundFiles = append(*foundFiles, ui.FileListItem{
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
					if len(*foundFiles) > 0 {
						fileOpPanel.Enable()
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