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

// Helper methods
func (sl *StructuredLogger) Info(message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	sl.Log("INFO", message, fields)
}

func (sl *StructuredLogger) Error(message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	sl.Log("ERROR", message, fields)
}

func (sl *StructuredLogger) Warn(message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	sl.Log("WARN", message, fields)
}

func (sl *StructuredLogger) Debug(message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	sl.Log("DEBUG", message, fields)
}

// Println maintains compatibility with existing log.Logger
func (sl *StructuredLogger) Println(v ...interface{}) {
	sl.Info(fmt.Sprint(v...), nil)
}

// Printf maintains compatibility with existing log.Logger
func (sl *StructuredLogger) Printf(format string, v ...interface{}) {
	sl.Info(fmt.Sprintf(format, v...), nil)
}
