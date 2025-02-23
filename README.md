# Koe no Search
<p align="center">
  <img width="160" height="160" src="https://github.com/AlestackOverglow/koe-no-search/raw/main/koe.png">
</p>

<div align="center">

> 🔍 Lightning-fast file search utility with modern GUI and CLI interfaces, designed for efficiency and ease of use.
> 
> If you find this project helpful, please consider giving it a star ⭐ It helps others discover the project and motivates further development.

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey.svg)]()

</div>

## Table of Contents
- [Quick Start](#quick-start)
- [Key Features](#key-features)
- [System Requirements](#system-requirements)
- [Installation](#installation)
- [Usage Examples](#usage-examples)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [License](#license)
- [Acknowledgments](#acknowledgments)

## Quick Start

1. Download the latest release for your platform
2. Run the executable:
   - GUI version: `koe-no-search-gui`
   - CLI version: `koe-no-search-cli -p "*.txt" /path/to/search`
3. Start searching!
<img src="screenshot.png" width="401" height="314">

## Key Features

- Instant search without indexing
- Modern, intuitive GUI and CLI interfaces
- High-performance concurrent search
- Cross-platform support
- Built-in file operations
- Real-time progress tracking
- Pattern and extension filtering
- Case-sensitive/insensitive search

## System Requirements

### Minimum Requirements
- RAM: 256MB
- CPU: Dual-core processor
- Disk Space: 40MB
- OS: Windows 7+, Ubuntu 18.04+, macOS 10.13+

### Recommended
- RAM: 512MB+
- CPU: Quad-core processor
- OS: Latest version of your platform

## Installation

### Build Requirements
- Go 1.21 or later
- Git (for development)
- GCC compiler
- Platform-specific GUI dependencies:
  ```bash
  # Windows: MinGW-w64/TDM-GCC (with libgcc, libstdc++, libwinpthread)
  
  # Ubuntu/Debian
  sudo apt-get install gcc libgl1-mesa-dev xorg-dev
  
  # Fedora
  sudo dnf install gcc libXcursor-devel libXrandr-devel mesa-libGL-devel libXi-devel libXinerama-devel libXxf86vm-devel
  
  # macOS
  xcode-select --install
  brew install pkg-config
  ```

### Building from Source

1. Clone and prepare:
```bash
git clone https://github.com/AlestackOverglow/koe-no-search.git
cd koe-no-search
go mod download
```

2. Build the project:
```bash
# GUI version
go build -o koe-no-search-gui ./cmd/gui     # Linux/macOS
go build -ldflags "-H windowsgui" -o koe-no-search-gui.exe ./cmd/gui  # Windows

# CLI version (all platforms)
go build -o koe-no-search-cli ./cmd/cli
```

## Usage Examples

### GUI Interface
- Enter search pattern (e.g., `*.txt`)
- Select directory
- Click "Start Search"
- Use file operations for found files

### CLI Interface
```bash
# Basic search
koe-no-search-cli -p "*.txt" /path/to/search

# Advanced search with options
koe-no-search-cli -i -p "*.doc*" -e "pdf,doc,txt" /path/to/search
```

## Documentation
- [About](ABOUT_EN.md) - Detailed description and technical details
- [API Documentation](API.md) - Integration guide for developers
- [Future Improvements](ABOUT_EN.md#future-improvements) - Planned features and enhancements

## Contributing

1. Keep Koe in app (that anime girl)
2. Fork the repository
3. Create feature branch (`git checkout -b feature/amazing-feature`)
4. Commit changes (`git commit -m 'Add amazing feature'`)
5. Push to branch (`git push origin feature/amazing-feature`)
6. Open Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Fyne](https://fyne.io/) - GUI toolkit
- [xxHash](https://github.com/cespare/xxhash) - Fast hashing
- [cobra](https://github.com/spf13/cobra) - CLI interface
