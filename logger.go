// Package logger provides a customizable logging utility for Go with support for various log levels,
// formats, console output and log rotation. The package includes functions for logging messages at
// TRACE, DEBUG, INFO, WARNING, ERROR and FATAL levels.
// Both text and JSON output formats are supported, as well as log file rotation configuration with
// maximum size, number of backups and retention period settings.
// The logger can output messages to both file and console with the ability to set minimum log levels
// for each output type.
// If logger initialization fails, an error message will be output to the console.
//
// Main features:
//   - Logging at various levels (TRACE, DEBUG, INFO, WARNING, ERROR, FATAL)
//   - Output format support: standard text and JSON
//   - Log level configuration for file and console (can be set by string or number)
//   - Optional log rotation with compression of old files
//   - Colored console output for better visual perception
//
// Usage example:
//    config := logger.LogConfig{
//        FilePath: "./logs/app.log",
//        Format: "standard",
//        FileLevel: "debug",
//        ConsoleLevel: "info",
//        ConsoleOutput: true,
//        EnableRotation: true,
//        RotationConfig: logger.RotationConfig{
//            MaxSize: 10,
//            MaxBackups: 5,
//            MaxAge: 30,
//            Compress: true,
//        },
//    }
//
//    err := logger.InitLogger(config)
//    if err != nil {
//        fmt.Println("Logger initialization failed:", err)
//        return
//    }
//    logger.Info("Example informational message")
package logger

import (
    "encoding/json"
    "fmt"
    "io"
    "log"
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "time"

    "github.com/fatih/color"
    "github.com/natefinch/lumberjack"
)

// Global variable for logger instance
var logInstance *Logger

// LogConfig represents configuration settings for the logger.
type LogConfig struct {
    FilePath       string         // Full path to the log file
    Format         string         // Log format: "standard" or "json"
    FileLevel      interface{}    // Log level for file output: can be string or number
    ConsoleLevel   interface{}    // Log level for console output: can be string or number
    ConsoleOutput  bool           // Whether to output logs to console
    EnableRotation bool           // Whether to enable log rotation
    RotationConfig RotationConfig // Settings for log rotation
}

// RotationConfig contains settings for log rotation
type RotationConfig struct {
    MaxSize    int  // Maximum size in megabytes before log rotation
    MaxBackups int  // Maximum number of old log files to keep
    MaxAge     int  // Maximum number of days to keep old log files
    Compress   bool // Whether to compress old log files
}

// Logger represents a customizable logger with various configuration options
type Logger struct {
    FileLogger      *log.Logger
    ConsoleLogger   *log.Logger
    Config          LogConfig
    FileLogLevel    int
    ConsoleLogLevel int
    LogLevelMap     map[string]int
}

// InitLogger initializes the logger and saves the instance in the global variable logInstance.
//
// Arguments:
//   - config (LogConfig): Logger configuration with settings for log level, format, file output, and rotation.
//
// Returns:
//   - error: Error if initialization failed, otherwise nil.
func InitLogger(config LogConfig) error {
    var err error
    logInstance, err = NewLogger(config)
    if err != nil {
        // Output error message to console
        fmt.Println("Logger initialization failed:", err)
    }
    return err
}

// setDefaults sets default values for the logger configuration.
func setDefaults(config *LogConfig) {
    if config.Format == "" {
        config.Format = "standard"
    }
    if config.FileLevel == nil {
        config.FileLevel = "warning"
    }
    if config.ConsoleLevel == nil {
        config.ConsoleLevel = "warning"
    }
    if config.RotationConfig.MaxSize == 0 {
        config.RotationConfig.MaxSize = 10 // 10 MB
    }
    if config.RotationConfig.MaxBackups == 0 {
        config.RotationConfig.MaxBackups = 7 // 7 backups
    }
    if config.RotationConfig.MaxAge == 0 {
        config.RotationConfig.MaxAge = 30 // 30 days
    }
}

// NewLogger creates and returns a new Logger instance with the specified configuration.
//
// Arguments:
//   - config (LogConfig): Logger configuration, including log level, output, and rotation settings.
//
// Returns:
//   - (*Logger): Pointer to the new Logger instance.
//   - error: Error if the configuration is invalid or the log file is inaccessible.
func NewLogger(config LogConfig) (*Logger, error) {
    // Set default values
    setDefaults(&config)

    l := &Logger{
        Config: config,
        LogLevelMap: map[string]int{
            "trace":   0,
            "debug":   1,
            "info":    2,
            "warning": 3,
            "error":   4,
            "fatal":   5,
        },
    }

    // Function to get the numeric value of the log level
    getLogLevel := func(level interface{}) (int, error) {
        switch v := level.(type) {
        case string:
            logLevel, ok := l.LogLevelMap[strings.ToLower(v)]
            if !ok {
                return 0, fmt.Errorf("invalid log level: %s", v)
            }
            return logLevel, nil
        case int:
            if v < 0 || v > 5 {
                return 0, fmt.Errorf("numeric log level out of range: %d (valid range is 0 to 5)", v)
            }
            return v, nil
        default:
            return 0, fmt.Errorf("invalid type for log level: %T", v)
        }
    }

    // Set log levels for file and console
    fileLevel, err := getLogLevel(config.FileLevel)
    if err != nil {
        fmt.Println("Invalid file log level:", err)
        return nil, fmt.Errorf("invalid file log level: %v", err)
    }
    l.FileLogLevel = fileLevel

    consoleLevel, err := getLogLevel(config.ConsoleLevel)
    if err != nil {
        fmt.Println("Invalid console log level:", err)
        return nil, fmt.Errorf("invalid console log level: %v", err)
    }
    l.ConsoleLogLevel = consoleLevel

    // Setup file logging if a path is specified
    if config.FilePath != "" {
        dir := filepath.Dir(config.FilePath)
        if _, err := os.Stat(dir); os.IsNotExist(err) {
            return nil, fmt.Errorf("log directory does not exist: %s", dir)
        }

        var fileWriter io.Writer
        if config.EnableRotation {
            fileWriter = &lumberjack.Logger{
                Filename:   config.FilePath,
                MaxSize:    config.RotationConfig.MaxSize,
                MaxBackups: config.RotationConfig.MaxBackups,
                MaxAge:     config.RotationConfig.MaxAge,
                Compress:   config.RotationConfig.Compress,
            }
        } else {
            file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
            if err != nil {
                return nil, fmt.Errorf("failed to open log file: %v", err)
            }
            fileWriter = file
        }

        l.FileLogger = log.New(fileWriter, "", 0)
    } else {
        l.FileLogger = nil // No file logger if FilePath is not set
    }

    // Setup console output
    if config.ConsoleOutput {
        l.ConsoleLogger = log.New(os.Stdout, "", 0)
    }

    return l, nil
}

// log is an internal method that logs messages with the specified level and arguments.
func (l *Logger) log(level string, v ...interface{}) {
    msgLevel, ok := l.LogLevelMap[level]
    if !ok {
        return
    }

    if msgLevel < l.FileLogLevel && msgLevel < l.ConsoleLogLevel {
        return
    }

    timestamp := time.Now().Format(time.RFC3339)
    pid := os.Getpid()

    // Get caller information
    _, file, line, ok := runtime.Caller(3)
    if !ok {
        file = "unknown"
        line = 0
    } else {
        // Trim the file path to the project level
        file = trimPathToProject(file)
    }

    prefix := fmt.Sprintf("[%s] [PID: %d] [%s:%d] [%s] ", timestamp, pid, file, line, strings.ToUpper(level))

    var logEntry string

    if strings.ToLower(l.Config.Format) == "json" {
        logData := map[string]interface{}{
            "timestamp": timestamp,
            "level":     level,
            "pid":       pid,
            "file":      file,
            "line":      line,
            "message":   fmt.Sprint(v...),
        }
        jsonBytes, _ := json.Marshal(logData)
        logEntry = string(jsonBytes)
    } else {
        logEntry = prefix + fmt.Sprint(v...)
    }

    // Log to file only if file logger is set and level meets the threshold
    if l.FileLogger != nil && msgLevel >= l.FileLogLevel {
        l.FileLogger.Println(logEntry)
    }

    // Log to console if enabled and level meets the threshold
    if l.Config.ConsoleOutput && msgLevel >= l.ConsoleLogLevel {
        colorFunc := color.New(color.FgWhite).SprintFunc()
        switch level {
        case "trace":
            colorFunc = color.New(color.FgCyan).SprintFunc()
        case "debug":
            colorFunc = color.New(color.FgBlue).SprintFunc()
        case "info":
            colorFunc = color.New(color.FgGreen).SprintFunc()
        case "warning":
            colorFunc = color.New(color.FgYellow).SprintFunc()
        case "error":
            colorFunc = color.New(color.FgRed).SprintFunc()
        case "fatal":
            colorFunc = color.New(color.FgHiRed).SprintFunc()
        }
        l.ConsoleLogger.Println(colorFunc(logEntry))
    }
}

// trimPathToProject trims the file path to the project level.
func trimPathToProject(filePath string) string {
    // Assume the project directory is the one containing the "go.mod" file
    projectDir := findProjectDir()
    if projectDir == "" {
        return filepath.Base(filePath)
    }
    relPath, err := filepath.Rel(projectDir, filePath)
    if err != nil {
        return filepath.Base(filePath)
    }
    return relPath
}

// findProjectDir finds the project directory by looking for the "go.mod" file.
func findProjectDir() string {
    dir, err := os.Getwd()
    if err != nil {
        return ""
    }
    for {
        if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
            return dir
        }
        parentDir := filepath.Dir(dir)
        if parentDir == dir {
            break
        }
        dir = parentDir
    }
    return ""
}

// GetLoggerConfig returns the current logger configuration.
//
// Returns:
//   - (LogConfig): Logger configuration used in logInstance.
func GetLoggerConfig() LogConfig {
    if logInstance != nil {
        return logInstance.Config
    }
    return LogConfig{}
}

// Package-level wrapper functions for logger methods

// Trace logs a message at the TRACE level if the log level allows it.
//
// Arguments:
//   - v (...interface{}): Message to log.
func Trace(v ...interface{}) {
    if logInstance != nil {
        logInstance.Trace(v...)
    }
}

// Debug logs a message at the DEBUG level if the log level allows it.
//
// Arguments:
//   - v (...interface{}): Message to log.
func Debug(v ...interface{}) {
    if logInstance != nil {
        logInstance.Debug(v...)
    }
}

// Info logs a message at the INFO level if the log level allows it.
//
// Arguments:
//   - v (...interface{}): Message to log.
func Info(v ...interface{}) {
    if logInstance != nil {
        logInstance.Info(v...)
    }
}

// Warning logs a message at the WARNING level if the log level allows it.
//
// Arguments:
//   - v (...interface{}): Message to log.
func Warning(v ...interface{}) {
    if logInstance != nil {
        logInstance.Warning(v...)
    }
}

// Error logs a message at the ERROR level if the log level allows it.
//
// Arguments:
//   - v (...interface{}): Message to log.
func Error(v ...interface{}) {
    if logInstance != nil {
        logInstance.Error(v...)
    }
}

// Fatal logs a message at the FATAL level and terminates the application.
//
// Arguments:
//   - v (...interface{}): Message to log.
func Fatal(v ...interface{}) {
    if logInstance != nil {
        logInstance.Fatal(v...)
    }
}

// Tracef logs a formatted message at the TRACE level if the log level allows it.
//
// Arguments:
//   - format (string): Format string.
//   - v (...interface{}): Values for formatting the message.
func Tracef(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Tracef(format, v...)
    }
}

// Debugf logs a formatted message at the DEBUG level if the log level allows it.
//
// Arguments:
//   - format (string): Format string.
//   - v (...interface{}): Values for formatting the message.
func Debugf(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Debugf(format, v...)
    }
}

// Infof logs a formatted message at the INFO level if the log level allows it.
//
// Arguments:
//   - format (string): Format string.
//   - v (...interface{}): Values for formatting the message.
func Infof(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Infof(format, v...)
    }
}

// Warningf logs a formatted message at the WARNING level if the log level allows it.
//
// Arguments:
//   - format (string): Format string.
//   - v (...interface{}): Values for formatting the message.
func Warningf(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Warningf(format, v...)
    }
}

// Errorf logs a formatted message at the ERROR level if the log level allows it.
//
// Arguments:
//   - format (string): Format string.
//   - v (...interface{}): Values for formatting the message.
func Errorf(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Errorf(format, v...)
    }
}

// Fatalf logs a formatted message at the FATAL level and terminates the application.
//
// Arguments:
//   - format (string): Format string.
//   - v (...interface{}): Values for formatting the message.
func Fatalf(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Fatalf(format, v...)
        os.Exit(1)
    }
}

// Traceln logs a message at the TRACE level with a newline if the log level allows it.
//
// Arguments:
//   - v (...interface{}): Message to log.
func Traceln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Traceln(v...)
    }
}

// Debugln logs a message at the DEBUG level with a newline if the log level allows it.
//
// Arguments:
//   - v (...interface{}): Message to log.
func Debugln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Debugln(v...)
    }
}

// Infoln logs a message at the INFO level with a newline if the log level allows it.
//
// Arguments:
//   - v (...interface{}): Message to log.
func Infoln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Infoln(v...)
    }
}

// Warningln logs a message at the WARNING level with a newline if the log level allows it.
//
// Arguments:
//   - v (...interface{}): Message to log.
func Warningln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Warningln(v...)
    }
}

// Errorln logs a message at the ERROR level with a newline if the log level allows it.
//
// Arguments:
//   - v (...interface{}): Message to log.
func Errorln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Errorln(v...)
    }
}

// Fatalln logs a message at the FATAL level with a newline and terminates the application.
//
// Arguments:
//   - v (...interface{}): Message to log.
func Fatalln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Fatalln(v...)
        os.Exit(1)
    }
}

// Instance methods for the logger

// Trace logs a message at the TRACE level.
//
// Arguments:
//   - v (...interface{}): Message to log.
func (l *Logger) Trace(v ...interface{}) {
    l.log("trace", v...)
}

// Debug logs a message at the DEBUG level.
//
// Arguments:
//   - v (...interface{}): Message to log.
func (l *Logger) Debug(v ...interface{}) {
    l.log("debug", v...)
}

// Info logs a message at the INFO level.
//
// Arguments:
//   - v (...interface{}): Message to log.
func (l *Logger) Info(v ...interface{}) {
    l.log("info", v...)
}

// Warning logs a message at the WARNING level.
//
// Arguments:
//   - v (...interface{}): Message to log.
func (l *Logger) Warning(v ...interface{}) {
    l.log("warning", v...)
}

// Error logs a message at the ERROR level.
//
// Arguments:
//   - v (...interface{}): Message to log.
func (l *Logger) Error(v ...interface{}) {
    l.log("error", v...)
}

// Fatal logs a message at the FATAL level and terminates the application.
//
// Arguments:
//   - v (...interface{}): Message to log.
func (l *Logger) Fatal(v ...interface{}) {
    l.log("fatal", v...)
    os.Exit(1)
}

// Tracef logs a formatted message at the TRACE level.
//
// Arguments:
//   - format (string): Format string.
//   - v (...interface{}): Values for formatting the message.
func (l *Logger) Tracef(format string, v ...interface{}) {
    l.log("trace", fmt.Sprintf(format, v...))
}

// Debugf logs a formatted message at the DEBUG level.
//
// Arguments:
//   - format (string): Format string.
//   - v (...interface{}): Values for formatting the message.
func (l *Logger) Debugf(format string, v ...interface{}) {
    l.log("debug", fmt.Sprintf(format, v...))
}

// Infof logs a formatted message at the INFO level.
//
// Arguments:
//   - format (string): Format string.
//   - v (...interface{}): Values for formatting the message.
func (l *Logger) Infof(format string, v ...interface{}) {
    l.log("info", fmt.Sprintf(format, v...))
}

// Warningf logs a formatted message at the WARNING level.
//
// Arguments:
//   - format (string): Format string.
//   - v (...interface{}): Values for formatting the message.
func (l *Logger) Warningf(format string, v ...interface{}) {
    l.log("warning", fmt.Sprintf(format, v...))
}

// Errorf logs a formatted message at the ERROR level.
//
// Arguments:
//   - format (string): Format string.
//   - v (...interface{}): Values for formatting the message.
func (l *Logger) Errorf(format string, v ...interface{}) {
    l.log("error", fmt.Sprintf(format, v...))
}

// Fatalf logs a formatted message at the FATAL level and terminates the application.
//
// Arguments:
//   - format (string): Format string.
//   - v (...interface{}): Values for formatting the message.
func (l *Logger) Fatalf(format string, v ...interface{}) {
    l.log("fatal", fmt.Sprintf(format, v...))
    os.Exit(1)
}

// Traceln logs a message at the TRACE level with a newline.
//
// Arguments:
//   - v (...interface{}): Message to log.
func (l *Logger) Traceln(v ...interface{}) {
    l.log("trace", fmt.Sprintln(v...))
}

// Debugln logs a message at the DEBUG level with a newline.
//
// Arguments:
//   - v (...interface{}): Message to log.
func (l *Logger) Debugln(v ...interface{}) {
    l.log("debug", fmt.Sprintln(v...))
}

// Infoln logs a message at the INFO level with a newline.
//
// Arguments:
//   - v (...interface{}): Message to log.
func (l *Logger) Infoln(v ...interface{}) {
    l.log("info", fmt.Sprintln(v...))
}

// Warningln logs a message at the WARNING level with a newline.
//
// Arguments:
//   - v (...interface{}): Message to log.
func (l *Logger) Warningln(v ...interface{}) {
    l.log("warning", fmt.Sprintln(v...))
}

// Errorln logs a message at the ERROR level with a newline if the log level allows it.
//
// Arguments:
//   - v (...interface{}): Message to log.
func (l *Logger) Errorln(v ...interface{}) {
    l.log("error", fmt.Sprintln(v...))
}

// Fatalln logs a message at the FATAL level with a newline and terminates the application.
//
// Arguments:
//   - v (...interface{}): Message to log.
func (l *Logger) Fatalln(v ...interface{}) {
    l.log("fatal", fmt.Sprintln(v...))
    os.Exit(1)
}