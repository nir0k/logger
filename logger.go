// Package logger provides a customizable logging utility with support for different log levels,
// formats, console output, and log rotation.
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

// Global variable for the logger instance
var logInstance *Logger

// InitLogger initializes the logger and saves the instance in logInstance.
func InitLogger(config LogConfig) error {
    var err error
    logInstance, err = NewLogger(config)
    if err != nil {
        // Вывод сообщения об ошибке в консоль
        fmt.Println("Logger initialization failed:", err)
    }
    return err
}

// LogConfig represents the configuration settings for the Logger.
type LogConfig struct {
    FilePath       string         // Full path to the log file.
    Format         string         // Log format: "standard" or "json".
    FileLevel      interface{}    // Log level for file output: can be string or int.
    ConsoleLevel   interface{}    // Log level for console output: can be string or int.
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

// NewLogger creates a new Logger instance with the specified configuration.
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

    // Функция для получения числового значения уровня логирования
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

    // Устанавливаем уровни логирования для файла и консоли
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

    // Настройка логирования в файл, если указан путь
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

    // Настройка вывода на консоль
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

    if msgLevel < l.FileLogLevel && msgLevel < l.ConsoleLogLevel {
        return
    }

    timestamp := time.Now().Format(time.RFC3339)
    pid := os.Getpid()

    // Get the caller information
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

    // Log to file only if the file logger is set and the level meets the threshold
    if l.FileLogger != nil && msgLevel >= l.FileLogLevel {
        l.FileLogger.Println(logEntry)
    }

    // Log to console if enabled and the level meets the threshold
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
    // Assuming the project directory is the one containing the "go.mod" file
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
func GetLoggerConfig() LogConfig {
    if logInstance != nil {
        return logInstance.Config
    }
    return LogConfig{}
}

// Пакетные функции-обёртки для методов логгера

// Trace logs a message at the TRACE level.
func Trace(v ...interface{}) {
    if logInstance != nil {
        logInstance.Trace(v...)
    }
}

// Debug logs a message at the DEBUG level.
func Debug(v ...interface{}) {
    if logInstance != nil {
        logInstance.Debug(v...)
    }
}

// Info logs a message at the INFO level.
func Info(v ...interface{}) {
    if logInstance != nil {
        logInstance.Info(v...)
    }
}

// Warning logs a message at the WARNING level.
func Warning(v ...interface{}) {
    if logInstance != nil {
        logInstance.Warning(v...)
    }
}

// Error logs a message at the ERROR level.
func Error(v ...interface{}) {
    if logInstance != nil {
        logInstance.Error(v...)
    }
}

// Fatal logs a message at the FATAL level and exits the application.
func Fatal(v ...interface{}) {
    if logInstance != nil {
        logInstance.Fatal(v...)
    }
}

// Tracef logs a formatted message at the TRACE level.
func Tracef(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Tracef(format, v...)
    }
}

// Debugf logs a formatted message at the DEBUG level.
func Debugf(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Debugf(format, v...)
    }
}

// Infof logs a formatted message at the INFO level.
func Infof(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Infof(format, v...)
    }
}

// Warningf logs a formatted message at the WARNING level.
func Warningf(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Warningf(format, v...)
    }
}

// Errorf logs a formatted message at the ERROR level.
func Errorf(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Errorf(format, v...)
    }
}

// Fatalf logs a formatted message at the FATAL level and exits the application.
func Fatalf(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Fatalf(format, v...)
    }
}

// Traceln logs a message at the TRACE level with a newline.
func Traceln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Traceln(v...)
    }
}

// Debugln logs a message at the DEBUG level with a newline.
func Debugln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Debugln(v...)
    }
}

// Infoln logs a message at the INFO level with a newline.
func Infoln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Infoln(v...)
    }
}

// Warningln logs a message at the WARNING level with a newline.
func Warningln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Warningln(v...)
    }
}

// Errorln logs a message at the ERROR level with a newline.
func Errorln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Errorln(v...)
    }
}

// Fatalln logs a message at the FATAL level with a newline and exits the application.
func Fatalln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Fatalln(v...)
    }
}

// Методы экземпляра логгера

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
    os.Exit(1)
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