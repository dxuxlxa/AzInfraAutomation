package utils

import (
	"fmt"
	"os"
)

// PrintExit logs an error message and exits the program if an error occurred
func PrintExit(err error, message string) {
	if err != nil {
		ErrorLogger.Printf("%s: %v", message, err)
		fmt.Printf("%s: %v\n", message, err)
		os.Exit(1)
	} else {
		InfoLogger.Printf("%s: Success", message)
	}
}

// LogError logs an error without exiting
func LogError(err error, message string) error {
	if err != nil {
		ErrorLogger.Printf("%s: %v", message, err)
		return fmt.Errorf("%s: %v", message, err)
	}
	InfoLogger.Printf("%s: Success", message)
	return nil
}
