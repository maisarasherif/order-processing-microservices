package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type StructuredLogger struct {
	ServiceName string
	Environment string
}

func NewLogger(serviceName string) *StructuredLogger {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	return &StructuredLogger{
		ServiceName: serviceName,
		Environment: env,
	}
}

func (sl *StructuredLogger) Log(level string, message string, fields map[string]interface{}) {
	hostname, _ := os.Hostname()

	logEntry := make(map[string]interface{}, len(fields)+7)
	logEntry["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	logEntry["service_name"] = sl.ServiceName
	logEntry["level"] = level
	logEntry["message"] = message
	logEntry["env"] = sl.Environment
	logEntry["host"] = hostname
	logEntry["pid"] = os.Getpid()

	for k, v := range fields {
		logEntry[k] = v
	}

	if err := json.NewEncoder(os.Stdout).Encode(logEntry); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode log entry: %v\n", err)
	}
}

// Info logs informational messages
func (sl *StructuredLogger) Info(message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	sl.Log("INFO", message, fields)
}

// Error logs error messages
func (sl *StructuredLogger) Error(message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	sl.Log("ERROR", message, fields)
}

// Warn logs warning messages
func (sl *StructuredLogger) Warn(message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	sl.Log("WARN", message, fields)
}

// Debug logs debug messages
func (sl *StructuredLogger) Debug(message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	sl.Log("DEBUG", message, fields)
}

// Println maintains compatibility with log.Logger
func (sl *StructuredLogger) Println(v ...interface{}) {
	sl.Info(fmt.Sprint(v...), nil)
}

// Printf maintains compatibility with log.Logger
func (sl *StructuredLogger) Printf(format string, v ...interface{}) {
	sl.Info(fmt.Sprintf(format, v...), nil)
}

// Print maintains compatibility with log.Logger
func (sl *StructuredLogger) Print(v ...interface{}) {
	sl.Info(fmt.Sprint(v...), nil)
}

// Fatal logs an error and exits with status 1
func (sl *StructuredLogger) Fatal(v ...interface{}) {
	sl.Error(fmt.Sprint(v...), nil)
	os.Exit(1)
}

// Fatalf logs a formatted error and exits with status 1
func (sl *StructuredLogger) Fatalf(format string, v ...interface{}) {
	sl.Error(fmt.Sprintf(format, v...), nil)
	os.Exit(1)
}

// Fatalln logs an error with newline and exits with status 1
func (sl *StructuredLogger) Fatalln(v ...interface{}) {
	sl.Error(fmt.Sprint(v...), nil)
	os.Exit(1)
}

// Panic logs a message and panics
func (sl *StructuredLogger) Panic(v ...interface{}) {
	msg := fmt.Sprint(v...)
	sl.Error(msg, nil)
	panic(msg)
}

// Panicf logs a formatted message and panics
func (sl *StructuredLogger) Panicf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	sl.Error(msg, nil)
	panic(msg)
}

// Panicln logs a message with newline and panics
func (sl *StructuredLogger) Panicln(v ...interface{}) {
	msg := fmt.Sprint(v...)
	sl.Error(msg, nil)
	panic(msg)
}
