// Package logger provides a simple logger for terragrunt-ls.
package logger

import (
	"log"
	"os"
)

// BuildLogger builds the standard logger for terragrunt-ls.
//
// When supplied with a filename, it'll create a new file and write logs to it.
// Otherwise, it'll write logs to stderr.
func BuildLogger(filename string) *log.Logger {
	if filename == "" {
		return log.New(os.Stderr, "[terragrunt-ls] ", log.Ldate|log.Ltime|log.Lshortfile)
	}

	const globalReadWrite = 0666

	logfile, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, globalReadWrite)
	if err != nil {
		panic("Failed to open log file: " + err.Error())
	}

	return log.New(logfile, "[terragrunt-ls] ", log.Ldate|log.Ltime|log.Lshortfile)
}

// BuildTestLogger builds a logger for testing purposes.
func BuildTestLogger() *log.Logger {
	return BuildLogger("")
}
