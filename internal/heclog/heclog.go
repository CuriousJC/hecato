package heclog

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func LogSetup(buildContext string) (*os.File, error) {
	logFilePath := ""
	if buildContext == "development" {
		logFilePath = filepath.Join("c:/repos/hecato", "app.log")
	} else {
		// Get the path to the executable directory
		execDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatalf("failed to get executable directory: %v", err)
		}
		// Create or open the log file in the executable directory
		logFilePath = filepath.Join(execDir, "app.log")
	}

	println("Log Path ", logFilePath)
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	// Configure the log package to use the log file
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile) // Customize log format as needed
	log.Println("Logger starting...")

	return logFile, nil
}

// logMessage logs a message line and optionally prints to the console
func LogMessage(logToConsole bool, v ...interface{}) {
	log.Println(v...)
	if logToConsole {
		fmt.Println(v...)
	}
}

// logMessagef logs a formatted message and optionally prints to the console
func LogMessagef(logToConsole bool, format string, args ...interface{}) {
	log.Printf(format, args...)
	if logToConsole {
		fmt.Printf(format, args...)
	}
}
