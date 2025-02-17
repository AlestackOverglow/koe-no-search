package explorer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	
	"fyne.io/fyne/v2/dialog"
	
	"filesearch/internal/search"
)

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
		// Convert path to Windows format
		absPath = strings.ReplaceAll(absPath, "/", "\\")
		
		// Use shell command to open explorer
		cmdPath := os.Getenv("COMSPEC")
		if cmdPath == "" {
			cmdPath = `C:\Windows\System32\cmd.exe`
		}
		
		// Use /c to close cmd after execution and start to run explorer asynchronously
		cmd := exec.Command(cmdPath, "/c", "start", "explorer.exe", "/select,", absPath)
		if err := cmd.Run(); err != nil {
			search.LogError("Failed to open in explorer: %v", err)
			dialog.ShowError(fmt.Errorf("Failed to open in explorer: %v", err), nil)
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

// GetAllDrives returns a list of all available drives in Windows/Linux
func GetAllDrives() []string {
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