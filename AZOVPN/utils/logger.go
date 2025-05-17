package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
	logFile     *os.File
)

// InitLogger initializes the logging system
func InitLogger() error {
	// Create logs directory if it doesn't exist
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logDir, fmt.Sprintf("azure_ovpn_%s.log", timestamp))

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	logFile = file

	// Set up loggers
	InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Also write to stdout
	InfoLogger.SetOutput(os.Stdout)
	ErrorLogger.SetOutput(os.Stderr)

	InfoLogger.Printf("Logging initialized, writing to %s", logPath)
	return nil
}

// CloseLogger closes the log file
func CloseLogger() {
	if logFile != nil {
		logFile.Close()
	}
}

// LogAndExit logs an error and exits the program
func LogAndExit(err error, message string) {
	if err != nil {
		ErrorLogger.Printf("%s: %v", message, err)
		os.Exit(1)
	}
}
