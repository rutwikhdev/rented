package utils

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type Logger struct {
	file *os.File
	mu   sync.Mutex
}

func NewLogger(path string) (*Logger, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{file: f}, nil
}

func (l *Logger) Info(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	timestamp := time.Now().Format(time.RFC3339)
	line := fmt.Sprintf("%s INFO %s", timestamp, msg)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			line += fmt.Sprintf(" %v=%v", args[i], args[i+1])
		}
	}
	fmt.Fprintln(l.file, line)
}

func (l *Logger) Error(msg string, err error, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	timestamp := time.Now().Format(time.RFC3339)
	line := fmt.Sprintf("%s ERROR %s", timestamp, msg)
	if err != nil {
		line += fmt.Sprintf(" err=%q", err.Error())
	}
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			line += fmt.Sprintf(" %v=%v", args[i], args[i+1])
		}
	}
	fmt.Fprintln(l.file, line)
}

func (l *Logger) Close() error {
	return l.file.Close()
}
