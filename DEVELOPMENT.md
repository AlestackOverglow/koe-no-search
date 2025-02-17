# Koe no Search - Development Guide

## Project Structure
```
koe-no-search/
├── cmd/
│   ├── cli/         # CLI application
│   │   └── main.go
│   └── gui/         # GUI application
│       └── main.go
├── internal/
│   └── search/      # Core search functionality
│       ├── assets/  # Embedded resources
│       │   └── icon.png
│       ├── cache.go
│       ├── globals.go
│       ├── logger.go
│       ├── matcher.go
│       ├── priority.go
│       ├── process.go
│       ├── processor.go
│       ├── resources.go
│       ├── search.go
│       ├── types.go
│       └── walker.go
├── .gitignore
├── API.md          # API documentation
├── DEVELOPMENT.md  # This file
├── LICENSE
├── README.md
└── go.mod
```

## Build Commands

### Prerequisites
```bash
# Install Go 1.21 or later
# Install GCC (MinGW-w64) for Windows
# Install required dependencies for Linux:
sudo apt-get install gcc libgl1-mesa-dev xorg-dev
```

### Building
```bash
# GUI version (Windows)
go build -ldflags "-X 'filesearch/internal/search.Version=0.2.0' -H windowsgui" -o koe-no-search-gui.exe ./cmd/gui

# GUI version (Linux/macOS)
go build -ldflags "-X 'filesearch/internal/search.Version=0.2.0'" -o koe-no-search-gui ./cmd/gui

# CLI version
go build -ldflags "-X 'filesearch/internal/search.Version=0.2.0'" -o koe-no-search-cli ./cmd/cli
```

## Git Commands

### Initial Setup
```bash
git init
git add .
git commit -m "Initial commit"
git branch -M main
git remote add origin https://github.com/AlestackOverglow/koe-no-search.git
git push -u origin main
```

### Regular Development Flow
```bash
# Check status
git status

# Add changes
git add .

# Create commit
git commit -m "feat: description of changes"

# Push changes
git push origin main
```

### Commit Message Prefixes
- feat: New features
- fix: Bug fixes
- docs: Documentation changes
- style: Code style changes
- refactor: Code refactoring
- test: Test changes
- chore: Build process or auxiliary tool changes

## Adding Resources

### Icons
1. Create directory for assets:
```bash
mkdir -p internal/search/assets
```

2. Add icon file:
```bash
# Copy icon.png to internal/search/assets/
```

3. Create/update resources.go:
```go
package search

import (
    _ "embed"
)

//go:embed assets/icon.png
var IconData []byte
```

4. Use in GUI (cmd/gui/main.go):
```go
if len(search.IconData) > 0 {
    iconResource := fyne.NewStaticResource("icon.png", search.IconData)
    a.SetIcon(iconResource)
}
```

### Background Images
1. Add image to assets:
```bash
# Copy background.png to internal/search/assets/
```

2. Update resources.go:
```go
//go:embed assets/background.png
var BackgroundData []byte
```

3. Use in GUI:
```go
backgroundResource := fyne.NewStaticResource("background.png", search.BackgroundData)
backgroundImage := canvas.NewImageFromResource(backgroundResource)
backgroundImage.Resize(fyne.NewSize(800, 600))
backgroundImage.FillMode = canvas.ImageFillStretch
```

## Version Management

### Version Information
Version information is stored in internal/search/globals.go:
```go
var (
    Version = "0.2.0"
    BuildTime = "unknown"
    GitCommit = "unknown"
)
```

### Building with Version Info
```bash
go build -ldflags "-X 'filesearch/internal/search.Version=0.2.0' -X 'filesearch/internal/search.BuildTime=$(date)' -X 'filesearch/internal/search.GitCommit=$(git rev-parse HEAD)'" -o koe-no-search-gui.exe ./cmd/gui
```

## GUI Development

### Main Window Layout
```go
content := container.NewHSplit(
    inputs,  // Left panel with controls
    container.NewScroll(resultsList),  // Right panel with results
)
```

### File Operations Section
```go
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

fileOpAccordion := widget.NewAccordion(
    widget.NewAccordionItem("File Operations", fileOpContent),
)
```

## Search Implementation

### Basic Search
```go
opts := search.SearchOptions{
    RootDirs:   []string{"/path"},
    Patterns:   []string{"*.txt"},
    MaxWorkers: runtime.NumCPU(),
    IgnoreCase: true,
}

results := search.Search(opts)
```

### File Operations
```go
fileOp := search.FileOperationOptions{
    Operation: search.CopyFiles,
    TargetDir: targetDir,
    ConflictPolicy: search.Skip,
}
```

## Debugging

### Logger Setup
```go
search.InitLogger()
defer search.CloseLogger()
```

### Log Levels
```go
search.LogDebug("Debug message")
search.LogInfo("Info message")
search.LogWarning("Warning message")
search.LogError("Error message")
```

## Common Issues and Solutions

### Windows Explorer Integration
Problem: Explorer opens multiple times
Solution: Use cmd.exe to open explorer:
```go
cmd := exec.Command(cmdPath, "/c", "start", "explorer.exe", "/select,", absPath)
```

### Memory Management
Problem: High memory usage
Solution: Adjust buffer sizes:
```go
opts.BatchSize = 50
opts.BufferSize = 500
opts.UseMMap = false
```

### Performance
Problem: Slow search
Solution: Increase workers and use mmap:
```go
opts.MaxWorkers = runtime.NumCPU() * 2
opts.UseMMap = true
opts.MinMMapSize = 1024 * 1024
```

## Testing

### Manual Testing Steps
1. Test basic search functionality
2. Test file operations (copy, move, delete)
3. Test with different file sizes
4. Test with different search patterns
5. Test error handling
6. Test GUI responsiveness
7. Test memory usage
8. Test cancellation 