// Package logger provides a customizable logging utility with support for different log levels,
// formats, console output, and log rotation.
package logger

import (
    "encoding/json"
    "fmt"
    "io"
    "log"
    "os"
    "strings"
    "time"

    "github.com/fatih/color"
    "github.com/natefinch/lumberjack"
)

// LogConfig represents the configuration settings for the Logger.
type LogConfig struct {
    Directory      string         // Directory where log files will be stored.
    Format         string         // Log format: "standard" or "json".
    FileLevel      string         // Minimum log level for file output: "trace", "debug", "info", "warning", "error", "fatal".
    ConsoleLevel   string         // Minimum log level for console output: "trace", "debug", "info", "warning", "error", "fatal".
    ConsoleOutput  bool           // Whether to output logs to the console.
    EnableRotation bool           // Whether to enable log rotation.
    RotationConfig RotationConfig // Settings for log rotation.
}

// RotationConfig contains settings for log rotation.
type RotationConfig struct {
    MaxSize    int  // Maximum size in megabytes before log rotation.
    MaxBackups int  // Maximum number of old log files to retain.
    MaxAge     int  // Maximum number of days to retain old log files.
    Compress   bool // Whether to compress rotated log files.
}

// Logger represents a customizable logger with various configuration options.
type Logger struct {
    FileLogger      *log.Logger
    ConsoleLogger   *log.Logger
    Config          LogConfig
    FileLogLevel    int
    ConsoleLogLevel int
    LogLevelMap     map[string]int
}

// setDefaults sets default values for the logger configuration.
func setDefaults(config *LogConfig) {
    if config.Directory == "" {
        config.Directory = "./logs"
    }
    if config.Format == "" {
        config.Format = "standard"
    }
    if config.FileLevel == "" {
        config.FileLevel = "info"
    }
    if config.ConsoleLevel == "" {
        config.ConsoleLevel = "info"
    }
    if config.RotationConfig.MaxSize == 0 {
        config.RotationConfig.MaxSize = 10 // 100 MB
    }
    if config.RotationConfig.MaxBackups == 0 {
        config.RotationConfig.MaxBackups = 7 // 7 backups
    }
    if config.RotationConfig.MaxAge == 0 {
        config.RotationConfig.MaxAge = 30 // 30 days
    }
}

// NewLogger creates a new Logger instance with the specified configuration.
// It initializes loggers for file output and console output based on the configuration.
//
// Returns an error if the log level is invalid or if there is an issue creating the log directory or files.
// NewLogger creates a new Logger instance based on the provided LogConfig.
// It sets up the log level, checks and creates the log directory if it does not exist,
// and configures file and console outputs for logging.
//
// Parameters:
//   - config: LogConfig struct containing the configuration for the logger.
//
// Returns:
//   - *Logger: A pointer to the created Logger instance.
//   - error: An error if the logger could not be created, otherwise nil.
//
// Possible errors:
//   - If the log level specified in the config is invalid.
//   - If the log directory could not be created.
//   - If the log file could not be opened.
func NewLogger(config LogConfig) (*Logger, error) {
    // Устанавливаем значения по умолчанию
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

    // Set the file log level
    fileLevel, ok := l.LogLevelMap[strings.ToLower(config.FileLevel)]
    if !ok {
        return nil, fmt.Errorf("invalid file log level: %s", config.FileLevel)
    }
    l.FileLogLevel = fileLevel

    // Set the console log level
    consoleLevel, ok := l.LogLevelMap[strings.ToLower(config.ConsoleLevel)]
    if !ok {
        return nil, fmt.Errorf("invalid console log level: %s", config.ConsoleLevel)
    }
    l.ConsoleLogLevel = consoleLevel

    // Check and create the log directory if it does not exist
    if _, err := os.Stat(config.Directory); os.IsNotExist(err) {
        err = os.MkdirAll(config.Directory, 0755)
        if err != nil {
            return nil, fmt.Errorf("failed to create log directory: %v", err)
        }
    }

    logFilePath := fmt.Sprintf("%s/log.txt", strings.TrimRight(config.Directory, "/"))

    // Setup file output
    var fileWriter io.Writer
    if config.EnableRotation {
        fileWriter = &lumberjack.Logger{
            Filename:   logFilePath,
            MaxSize:    config.RotationConfig.MaxSize,
            MaxBackups: config.RotationConfig.MaxBackups,
            MaxAge:     config.RotationConfig.MaxAge,
            Compress:   config.RotationConfig.Compress,
        }
    } else {
        file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err != nil {
            return nil, fmt.Errorf("failed to open log file: %v", err)
        }
        fileWriter = file
    }

    l.FileLogger = log.New(fileWriter, "", 0)

    // Setup console output
    if config.ConsoleOutput {
        l.ConsoleLogger = log.New(os.Stdout, "", 0)
    }

    return l, nil
}

// log is an internal method that logs messages with the given level and arguments.
func (l *Logger) log(level string, v ...interface{}) {
    msgLevel, ok := l.LogLevelMap[level]
    if !ok {
        return
    }

    timestamp := time.Now().Format(time.RFC3339)
    prefix := fmt.Sprintf("[%s] [%s] ", timestamp, strings.ToUpper(level))

    var logEntry string

    if strings.ToLower(l.Config.Format) == "json" {
        logData := map[string]interface{}{
            "timestamp": timestamp,
            "level":     level,
            "message":   fmt.Sprint(v...),
        }
        jsonBytes, _ := json.Marshal(logData)
        logEntry = string(jsonBytes)
    } else {
        logEntry = prefix + fmt.Sprint(v...)
    }

    // Log to file without color codes
    if l.FileLogger != nil && msgLevel >= l.FileLogLevel {
        l.FileLogger.Println(logEntry)
    }

    // Log to console with color codes
    if l.ConsoleLogger != nil && msgLevel >= l.ConsoleLogLevel {
        colorizedEntry := l.colorize(level, logEntry)
        l.ConsoleLogger.Println(colorizedEntry)
    }

    // Exit the program if the log level is fatal
    if level == "fatal" {
        os.Exit(1)
    }
}

// colorize applies color to the log message based on the log level.
func (l *Logger) colorize(level string, message string) string {
    switch strings.ToLower(level) {
    case "trace":
        return color.HiBlueString(message) // Light blue
    case "debug":
        return color.HiCyanString(message) // Light cyan
    case "info":
        return color.HiGreenString(message) // Light green
    case "warning":
        return color.HiYellowString(message) // Light yellow
    case "error":
        return color.HiRedString(message) // Light red
    case "fatal":
        return color.New(color.FgWhite, color.BgHiRed).Sprint(message) // White text on bright red background
    default:
        return message
    }
}

// Trace logs a message at the TRACE level.
func (l *Logger) Trace(v ...interface{}) {
    l.log("trace", v...)
}

// Debug logs a message at the DEBUG level.
func (l *Logger) Debug(v ...interface{}) {
    l.log("debug", v...)
}

// Info logs a message at the INFO level.
func (l *Logger) Info(v ...interface{}) {
    l.log("info", v...)
}

// Warning logs a message at the WARNING level.
func (l *Logger) Warning(v ...interface{}) {
    l.log("warning", v...)
}

// Error logs a message at the ERROR level.
func (l *Logger) Error(v ...interface{}) {
    l.log("error", v...)
}

// Fatal logs a message at the FATAL level and exits the application.
func (l *Logger) Fatal(v ...interface{}) {
    l.log("fatal", v...)
}

// Tracef logs a formatted message at the TRACE level.
func (l *Logger) Tracef(format string, v ...interface{}) {
    l.log("trace", fmt.Sprintf(format, v...))
}

// Debugf logs a formatted message at the DEBUG level.
func (l *Logger) Debugf(format string, v ...interface{}) {
    l.log("debug", fmt.Sprintf(format, v...))
}

// Infof logs a formatted message at the INFO level.
func (l *Logger) Infof(format string, v ...interface{}) {
    l.log("info", fmt.Sprintf(format, v...))
}

// Warningf logs a formatted message at the WARNING level.
func (l *Logger) Warningf(format string, v ...interface{}) {
    l.log("warning", fmt.Sprintf(format, v...))
}

// Errorf logs a formatted message at the ERROR level.
func (l *Logger) Errorf(format string, v ...interface{}) {
    l.log("error", fmt.Sprintf(format, v...))
}

// Fatalf logs a formatted message at the FATAL level and exits the application.
func (l *Logger) Fatalf(format string, v ...interface{}) {
    l.log("fatal", fmt.Sprintf(format, v...))
    os.Exit(1)
}

// Traceln logs a message at the TRACE level with a newline.
func (l *Logger) Traceln(v ...interface{}) {
    l.log("trace", fmt.Sprintln(v...))
}

// Debugln logs a message at the DEBUG level with a newline.
func (l *Logger) Debugln(v ...interface{}) {
    l.log("debug", fmt.Sprintln(v...))
}

// Infoln logs a message at the INFO level with a newline.
func (l *Logger) Infoln(v ...interface{}) {
    l.log("info", fmt.Sprintln(v...))
}

// Warningln logs a message at the WARNING level with a newline.
func (l *Logger) Warningln(v ...interface{}) {
    l.log("warning", fmt.Sprintln(v...))
}

// Errorln logs a message at the ERROR level with a newline.
func (l *Logger) Errorln(v ...interface{}) {
    l.log("error", fmt.Sprintln(v...))
}

// Fatalln logs a message at the FATAL level with a newline and exits the application.
func (l *Logger) Fatalln(v ...interface{}) {
    l.log("fatal", fmt.Sprintln(v...))
    os.Exit(1)
}