package logger_test

import (
    "bytes"
    "io"
    "os"
    "path/filepath"
    "strings"
    "testing"
    "time"

    "github.com/nir0k/logger"
)

func TestConsoleOutput(t *testing.T) {
    // Create a buffer to capture console output
    var consoleOutput bytes.Buffer

    // Save the original os.Stdout
    originalStdout := os.Stdout

    // Create a pipe to redirect stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    config := logger.LogConfig{
        Directory:      "./test_logs_console",
        Format:         "standard",
        FileLevel:      "debug",
        ConsoleLevel:   "info",
        ConsoleOutput:  true,
        EnableRotation: false,
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    // Start a goroutine to read from the pipe
    done := make(chan bool)
    go func() {
        io.Copy(&consoleOutput, r)
        done <- true
    }()

    // Log a test message
    log.Info("Test informational message")

    // Close the writer to finish the goroutine
    w.Close()
    <-done

    // Restore the original os.Stdout
    os.Stdout = originalStdout

    // Check the console output for the test message
    output := consoleOutput.String()
    if !strings.Contains(output, "Test informational message") {
        t.Errorf("Message not found in console output")
    }

    // Remove the test log directory
    os.RemoveAll("./test_logs_console")
}

func TestFileOutput(t *testing.T) {
    // Create a temporary directory for logs
    logDir, err := os.MkdirTemp("", "test_logs_file")
    if err != nil {
        t.Fatalf("Failed to create temporary directory: %v", err)
    }
    defer os.RemoveAll(logDir) // Clean up after the test

    logFile := filepath.Join(logDir, "log.txt")

    config := logger.LogConfig{
        Directory:      logDir,
        Format:         "standard",
        FileLevel:      "debug",
        ConsoleLevel:   "info",
        ConsoleOutput:  false,
        EnableRotation: false,
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    // Log a test message
    log.Info("Test informational message to file")

    // Read the contents of the log file
	data, err := os.ReadFile(logFile)
    if err != nil {
        t.Fatalf("Failed to read log file: %v", err)
    }

    content := string(data)
    if !strings.Contains(content, "Test informational message to file") {
        t.Errorf("Message not found in log file")
    }
}

func TestLogRotation(t *testing.T) {
    // Create a temporary directory for logs
    logDir, err := os.MkdirTemp("", "test_logs_rotation")
    if err != nil {
        t.Fatalf("Failed to create temporary directory: %v", err)
    }
    defer os.RemoveAll(logDir) // Clean up after the test

    logFileName := "log.txt"

    config := logger.LogConfig{
        Directory:      logDir,
        Format:         "standard",
        FileLevel:      "debug",
        ConsoleLevel:   "info",
        ConsoleOutput:  false,
        EnableRotation: true,
        RotationConfig: logger.RotationConfig{
            MaxSize:    1, // 1 MB
            MaxBackups: 2,
            MaxAge:     1, // 1 day
            Compress:   false,
        },
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    // Write multiple small messages to exceed MaxSize
    smallMessage := strings.Repeat("A", 1024*10) // 10 KB
    messagesToWrite := 110                       // 110 * 10 KB = 1.1 MB

    for i := 0; i < messagesToWrite; i++ {
        log.Info("Message number", i, smallMessage)
    }

    // Wait for rotation to occur
    time.Sleep(1 * time.Second)

    // Check for rotated log files
    files, err := os.ReadDir(logDir)
    if err != nil {
        t.Fatalf("Failed to read log directory: %v", err)
    }

    // Count the number of log files
    logFiles := 0
    for _, file := range files {
        if file.Name() == logFileName || strings.HasPrefix(file.Name(), strings.TrimSuffix(logFileName, ".txt")) {
            logFiles++
            t.Logf("Found file: %s (size: %d bytes)", file.Name(), fileInfoSize(logDir, file.Name()))
        }
    }

    if logFiles < 2 {
        t.Errorf("Log rotation did not occur, found files: %d", logFiles)
    }
}

func TestLogRotationWithCompression(t *testing.T) {
    // Create a temporary directory for logs
    logDir, err := os.MkdirTemp("", "test_logs_rotation_compress")
    if err != nil {
        t.Fatalf("Failed to create temporary directory: %v", err)
    }
    defer os.RemoveAll(logDir) // Clean up after the test

    config := logger.LogConfig{
        Directory:      logDir,
        Format:         "standard",
        FileLevel:      "debug",
        ConsoleLevel:   "info",
        ConsoleOutput:  false,
        EnableRotation: true,
        RotationConfig: logger.RotationConfig{
            MaxSize:    1,  // 1 MB
            MaxBackups: 2,
            MaxAge:     1,  // 1 day
            Compress:   true, // Enable compression
        },
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    // Write multiple small messages to exceed MaxSize
    smallMessage := strings.Repeat("A", 1024*10) // 10 KB
    messagesToWrite := 110                       // 110 * 10 KB = 1.1 MB

    for i := 0; i < messagesToWrite; i++ {
        log.Info("Message number", i, smallMessage)
    }

    // Wait for rotation and compression to occur
    time.Sleep(2 * time.Second)

    // Check for compressed log files
    files, err := os.ReadDir(logDir)
    if err != nil {
        t.Fatalf("Failed to read log directory: %v", err)
    }

    compressedFiles := 0
    for _, file := range files {
        if strings.HasSuffix(file.Name(), ".gz") {
            compressedFiles++
            t.Logf("Found compressed file: %s (size: %d bytes)", file.Name(), fileInfoSize(logDir, file.Name()))
        } else {
            t.Logf("Found file: %s (size: %d bytes)", file.Name(), fileInfoSize(logDir, file.Name()))
        }
    }

    if compressedFiles == 0 {
        t.Errorf("No compressed files found")
    } else {
        t.Logf("Number of compressed files: %d", compressedFiles)
    }
}

// fileInfoSize returns the size of a file in bytes.
func fileInfoSize(dir, name string) int64 {
    info, err := os.Stat(filepath.Join(dir, name))
    if err != nil {
        return 0
    }
    return info.Size()
}

func TestDefaultConfig(t *testing.T) {
    config := logger.LogConfig{}

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    // Проверяем значения по умолчанию
    if log.Config.Directory != "./logs" {
        t.Errorf("Expected default Directory to be './logs', got '%s'", log.Config.Directory)
    }
    if log.Config.Format != "standard" {
        t.Errorf("Expected default Format to be 'standard', got '%s'", log.Config.Format)
    }
    if log.Config.FileLevel != "info" {
        t.Errorf("Expected default FileLevel to be 'info', got '%s'", log.Config.FileLevel)
    }
    if log.Config.ConsoleLevel != "info" {
        t.Errorf("Expected default ConsoleLevel to be 'info', got '%s'", log.Config.ConsoleLevel)
    }
    if log.Config.RotationConfig.MaxSize != 10 {
        t.Errorf("Expected default RotationConfig.MaxSize to be 100, got %d", log.Config.RotationConfig.MaxSize)
    }
    if log.Config.RotationConfig.MaxBackups != 7 {
        t.Errorf("Expected default RotationConfig.MaxBackups to be 7, got %d", log.Config.RotationConfig.MaxBackups)
    }
    if log.Config.RotationConfig.MaxAge != 30 {
        t.Errorf("Expected default RotationConfig.MaxAge to be 30, got %d", log.Config.RotationConfig.MaxAge)
    }
}

func TestLogMethods(t *testing.T) {
    // Create a buffer to capture console output
    var consoleOutput bytes.Buffer

    // Save the original os.Stdout
    originalStdout := os.Stdout

    // Create a pipe to redirect stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    config := logger.LogConfig{
        Directory:      "./test_logs_methods",
        Format:         "standard",
        FileLevel:      "trace",
        ConsoleLevel:   "trace",
        ConsoleOutput:  true,
        EnableRotation: false,
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    // Start a goroutine to read from the pipe
    done := make(chan bool)
    go func() {
        io.Copy(&consoleOutput, r)
        done <- true
    }()

    // Log test messages
    log.Trace("TRACE level message")
    log.Debug("Debug message")
    log.Info("Informational message")
    log.Warning("Warning")
    log.Error("Error message")
    // log.Fatal("Critical error, application will terminate")

    log.Tracef("TRACE level message: %d", 1)
    log.Debugf("Debug message: %d", 2)
    log.Infof("Informational message: %d", 3)
    log.Warningf("Warning message: %d", 4)
    log.Errorf("Error message: %d", 5)
    // log.Fatalf("Critical error, application will terminate: %d", 6)

    log.Traceln("TRACE level message with newline")
    log.Debugln("Debug message with newline")
    log.Infoln("Informational message with newline")
    log.Warningln("Warning message with newline")
    log.Errorln("Error message with newline")
    // log.Fatalln("Critical error, application will terminate with newline")

    // Close the writer to finish the goroutine
    w.Close()
    <-done

    // Restore the original os.Stdout
    os.Stdout = originalStdout

    // Check the console output for the test messages
    output := consoleOutput.String()
    lines := strings.Split(output, "\n")

    for i, line := range lines {
        if i > 0 && line != "" && !strings.HasPrefix(line, "[") {
            t.Errorf("Expected new log entry to start with '[', got '%s'", line)
        }
    }

    if !strings.Contains(output, "TRACE level message: 1") {
        t.Errorf("Expected 'TRACE level message: 1' in output, got '%s'", output)
    }
    if !strings.Contains(output, "Debug message: 2") {
        t.Errorf("Expected 'Debug message: 2' in output, got '%s'", output)
    }
    if !strings.Contains(output, "Informational message: 3") {
        t.Errorf("Expected 'Informational message: 3' in output, got '%s'", output)
    }
    if !strings.Contains(output, "Warning message: 4") {
        t.Errorf("Expected 'Warning message: 4' in output, got '%s'", output)
    }
    if !strings.Contains(output, "Error message: 5") {
        t.Errorf("Expected 'Error message: 5' in output, got '%s'", output)
    }
    if !strings.Contains(output, "TRACE level message with newline") {
        t.Errorf("Expected 'TRACE level message with newline' in output, got '%s'", output)
    }
    if !strings.Contains(output, "Debug message with newline") {
        t.Errorf("Expected 'Debug message with newline' in output, got '%s'", output)
    }
    if !strings.Contains(output, "Informational message with newline") {
        t.Errorf("Expected 'Informational message with newline' in output, got '%s'", output)
    }
    if !strings.Contains(output, "Warning message with newline") {
        t.Errorf("Expected 'Warning message with newline' in output, got '%s'", output)
    }
    if !strings.Contains(output, "Error message with newline") {
        t.Errorf("Expected 'Error message with newline' in output, got '%s'", output)
    }

    // Remove the test log directory
    os.RemoveAll("./test_logs_methods")
}