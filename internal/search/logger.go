package search

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
)

type Logger struct {
	mu     sync.Mutex
	file   *os.File
	level  LogLevel
}

var (
	globalLogger *Logger
	loggerOnce   sync.Once
)

func initLogger() {
	loggerOnce.Do(func() {
		// Изменяем путь к лог-файлу на текущую директорию
		executable, err := os.Executable()
		if err != nil {
			fmt.Printf("Failed to get executable path: %v\n", err)
			return
		}
		
		logFile := filepath.Join(filepath.Dir(executable), "filesearch.log")
		
		// Выводим путь к лог-файлу
		fmt.Printf("Log file path: %s\n", logFile)
		
		// Проверим, существует ли файл
		if _, err := os.Stat(logFile); err != nil {
			fmt.Printf("Creating new log file: %s\n", logFile)
		}
		
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Failed to open log file: %v\n", err)
			return
		}
		
		globalLogger = &Logger{
			file:  file,
			level: ERROR,
		}
		
		// Записываем начальную информацию в лог
		fmt.Fprintf(file, "\n=== Log session started at %s ===\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Fprintf(file, "OS: %s\nArch: %s\nExecutable Dir: %s\nLog File: %s\n\n",
			runtime.GOOS, runtime.GOARCH, filepath.Dir(executable), logFile)
			
		fmt.Printf("Logger initialized successfully\n")
	})
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if l == nil || l.file == nil {
		fmt.Printf("Logger not initialized properly\n")
		return
	}
	
	if level < l.level {
		return
	}
	
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Get caller info
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}
	
	// Format timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	
	// Format level
	levelStr := "UNKNOWN"
	switch level {
	case DEBUG:
		levelStr = "DEBUG"
	case INFO:
		levelStr = "INFO"
	case WARNING:
		levelStr = "WARNING"
	case ERROR:
		levelStr = "ERROR"
	}
	
	// Format message
	msg := fmt.Sprintf(format, args...)
	
	// Write log entry
	logEntry := fmt.Sprintf("[%s] %s %s:%d: %s\n",
		timestamp, levelStr, filepath.Base(file), line, msg)
	
	// Пишем в файл и дублируем в консоль для отладки
	l.file.WriteString(logEntry)
	fmt.Print(logEntry)
}

func logDebug(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.log(DEBUG, format, args...)
	}
}

func logInfo(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.log(INFO, format, args...)
	}
}

func logWarning(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.log(WARNING, format, args...)
	}
}

func logError(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.log(ERROR, format, args...)
	}
}

func closeLogger() {
	if globalLogger != nil && globalLogger.file != nil {
		globalLogger.file.Close()
	}
}

func InitLogger() {
	initLogger()
}

func CloseLogger() {
	closeLogger()
}

func LogDebug(format string, args ...interface{}) {
	logDebug(format, args...)
}

func LogInfo(format string, args ...interface{}) {
	logInfo(format, args...)
}

func LogWarning(format string, args ...interface{}) {
	logWarning(format, args...)
}

func LogError(format string, args ...interface{}) {
	logError(format, args...)
} 