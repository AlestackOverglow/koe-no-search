package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"context"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/schollz/progressbar/v3"
	
	"filesearch/internal/search"
)

var (
	patterns        []string
	extensions      []string
	ignoreCase      bool
	workers         int
	bufferSize      int
	showSize        bool
	openInExplorer  bool
	showVersion     bool
)

// formatSize formats file size in human-readable form
func formatSize(size int64) string {
	switch {
	case size >= 1024*1024*1024:
		return fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
	case size >= 1024*1024:
		return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
	case size >= 1024:
		return fmt.Sprintf("%.2f KB", float64(size)/1024)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// openFileLocation opens file location in explorer
func openFileLocation(path string) error {
	path = filepath.Clean(path)
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "windows":
		cmdPath := os.Getenv("COMSPEC")
		if cmdPath == "" {
			cmdPath = `C:\Windows\System32\cmd.exe`
		}
		cmd = exec.Command(cmdPath, "/c", "explorer", "/select,", path)
	case "darwin":
		cmd = exec.Command("open", "-R", path)
	default: // Linux and other Unix-like systems
		cmd = exec.Command("xdg-open", filepath.Dir(path))
	}
	
	return cmd.Run()
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "filesearch [directories...]",
		Short: "Fast file search utility",
		Long: `A high-performance file search utility with advanced features.
Supports multiple patterns and extensions for searching.
Example: filesearch -p "*.txt" -p "*.doc" -e txt -e doc -i /home /usr`,
		Version: search.Version,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if showVersion {
				fmt.Printf("Koe no Search v%s\n", search.Version)
				fmt.Printf("Build Time: %s\n", search.BuildTime)
				fmt.Printf("Git Commit: %s\n", search.GitCommit)
				return
			}

			if workers <= 0 {
				workers = runtime.NumCPU()
			}

			// Create context with cancellation
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle Ctrl+C
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			stopChan := make(chan struct{})

			go func() {
				select {
				case <-sigChan:
					fmt.Println("\nSearch interrupted by user")
					close(stopChan)
					cancel()
				case <-ctx.Done():
					return
				}
			}()

			opts := search.SearchOptions{
				RootDirs:   args,
				Patterns:   patterns,
				Extensions: extensions,
				MaxWorkers: workers,
				IgnoreCase: ignoreCase,
				BufferSize: bufferSize,
				StopChan:   stopChan,
			}

			results := search.Search(opts)
			
			bar := progressbar.Default(-1, "Searching")
			
			count := 0
			foundFiles := make([]string, 0)
			
			// Process search results
			for result := range results {
				select {
				case <-ctx.Done():
					return
				default:
					bar.Add(1)
					if result.Error != nil {
						fmt.Printf("\nError processing %s: %v\n", result.Path, result.Error)
						continue
					}
					
					sizeStr := ""
					if showSize {
						sizeStr = fmt.Sprintf(" (%s)", formatSize(result.Size))
					}
					
					fmt.Printf("\nFound: %s%s\n", result.Path, sizeStr)
					foundFiles = append(foundFiles, result.Path)
					count++
				}
			}
			
			fmt.Printf("\nTotal files found: %d\n", count)
			
			// If only one file found and open in explorer option is enabled
			if openInExplorer && len(foundFiles) > 0 {
				fmt.Println("Opening file location...")
				if err := openFileLocation(foundFiles[0]); err != nil {
					fmt.Printf("Error opening file location: %v\n", err)
				}
			}
		},
	}

	rootCmd.Flags().StringSliceVarP(&patterns, "pattern", "p", []string{}, "Search patterns (can be specified multiple times)")
	rootCmd.Flags().StringSliceVarP(&extensions, "ext", "e", []string{}, "File extensions without dot (can be specified multiple times)")
	rootCmd.Flags().BoolVarP(&ignoreCase, "ignore-case", "i", false, "Ignore case")
	rootCmd.Flags().IntVarP(&workers, "workers", "w", 0, "Number of worker threads (default: number of CPU cores)")
	rootCmd.Flags().IntVarP(&bufferSize, "buffer", "b", 1000, "Size of the internal buffers")
	rootCmd.Flags().BoolVarP(&showSize, "size", "s", true, "Show file sizes")
	rootCmd.Flags().BoolVarP(&openInExplorer, "open", "o", false, "Open file location in explorer (when single file found)")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
} 