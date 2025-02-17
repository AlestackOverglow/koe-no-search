package search

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"bufio"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
)

// Оптимизированные константы для логгера
const (
	maxLogSize       = 10 * 1024 * 1024 // 10MB
	logBufferSize    = 32 * 1024        // 32KB
	maxLogRotations  = 5
)

type Logger struct {
	mu       sync.RWMutex
	writer   *bufio.Writer
	file     *os.File
	buffer   []byte
	disabled bool
}

var (
	globalLogger  *Logger
	loggerOnce   sync.Once
	loggerBuffer = make(chan string, 1000) // Буферизованный канал для логов
)

func init() {
	go processLogs() // Запускаем горутину для обработки логов
}

// processLogs обрабатывает логи асинхронно
func processLogs() {
	for msg := range loggerBuffer {
		if l := getLogger(); l != nil && !l.disabled {
			l.mu.Lock()
			if l.writer != nil {
				l.writer.WriteString(msg)
				// Периодически сбрасываем буфер
				if len(loggerBuffer) == 0 {
					l.writer.Flush()
				}
			}
			l.mu.Unlock()
		}
	}
}

// getLogger returns the global logger instance
func getLogger() *Logger {
	loggerOnce.Do(initLogger)
	return globalLogger
}

// initLogger initializes the global logger
func initLogger() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Failed to initialize logger: %v\n", r)
			globalLogger = &Logger{disabled: true}
		}
	}()

	logDir := filepath.Join(os.TempDir(), "koe-no-search-logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create log directory: %v", err))
	}

	logPath := filepath.Join(logDir, "search.log")
	rotateLogFile(logPath) // Ротация логов при инициализации

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file: %v", err))
	}

	writer := bufio.NewWriterSize(file, logBufferSize)
	globalLogger = &Logger{
		writer: writer,
		file:   file,
		buffer: make([]byte, 0, 1024),
	}

	// Write initial log entry
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(writer, "\n=== Log started at %s ===\n", timestamp)
	writer.Flush()
}

// rotateLogFile rotates log files if necessary
func rotateLogFile(logPath string) {
	if fi, err := os.Stat(logPath); err == nil {
		if fi.Size() > maxLogSize {
			// Rotate log files
			for i := maxLogRotations - 1; i > 0; i-- {
				oldPath := fmt.Sprintf("%s.%d", logPath, i)
				newPath := fmt.Sprintf("%s.%d", logPath, i+1)
				os.Rename(oldPath, newPath)
			}
			os.Rename(logPath, logPath+".1")
		}
	}
}

// logDebug writes a debug message to the log file
func logDebug(format string, args ...interface{}) {
	if l := getLogger(); l != nil && !l.disabled {
		msg := fmt.Sprintf("[DEBUG] "+format+"\n", args...)
		select {
		case loggerBuffer <- msg:
		default:
			// Если буфер полон, пропускаем сообщение
		}
	}
}

// logInfo writes an info message to the log file
func logInfo(format string, args ...interface{}) {
	if l := getLogger(); l != nil && !l.disabled {
		msg := fmt.Sprintf("[INFO] "+format+"\n", args...)
		select {
		case loggerBuffer <- msg:
		default:
			// Если буфер полон, пропускаем сообщение
		}
	}
}

// logError writes an error message to the log file
func logError(format string, args ...interface{}) {
	if l := getLogger(); l != nil && !l.disabled {
		msg := fmt.Sprintf("[ERROR] "+format+"\n", args...)
		// Для ошибок используем блокирующую запись
		loggerBuffer <- msg
	}
}

// Close closes the logger and flushes any pending writes
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.writer != nil {
		if err := l.writer.Flush(); err != nil {
			return fmt.Errorf("failed to flush log buffer: %v", err)
		}
	}

	if l.file != nil {
		if err := l.file.Sync(); err != nil {
			return fmt.Errorf("failed to sync log file: %v", err)
		}
		if err := l.file.Close(); err != nil {
			return fmt.Errorf("failed to close log file: %v", err)
		}
		l.file = nil
	}

	return nil
}

// Экспортируемые функции
func InitLogger() {
	initLogger()
}

func CloseLogger() {
	if l := getLogger(); l != nil {
		if err := l.Close(); err != nil {
			fmt.Printf("Failed to close logger: %v\n", err)
		}
	}
}

func LogDebug(format string, args ...interface{}) {
	logDebug(format, args...)
}

func LogInfo(format string, args ...interface{}) {
	logInfo(format, args...)
}

func LogWarning(format string, args ...interface{}) {
	if logger := getLogger(); logger != nil && !logger.disabled {
		msg := fmt.Sprintf("[WARNING] "+format+"\n", args...)
		select {
		case loggerBuffer <- msg:
		default:
			// Если буфер полон, пропускаем сообщение
		}
	}
}

func LogError(format string, args ...interface{}) {
	logError(format, args...)
} 