package logger_test

import (
    "bytes"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
    "testing"
    "time"

    "github.com/nir0k/logger"
)

func TestConsoleOutput(t *testing.T) {
    // Check logging only to console, without writing to file.
    var consoleOutput bytes.Buffer
    originalStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    config := logger.LogConfig{
        FilePath:      "", // File path not specified
        Format:        "standard",
        FileLevel:     "debug",
        ConsoleLevel:  "info",
        ConsoleOutput: true,
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    done := make(chan bool)
    go func() {
        io.Copy(&consoleOutput, r)
        done <- true
    }()

    // Log a message and check that it appears in the console
    log.Info("Test informational message")

    // Finish writing to console
    w.Close()
    <-done
    os.Stdout = originalStdout

    // Check console output
    output := consoleOutput.String()
    if !strings.Contains(output, "Test informational message") {
        t.Errorf("Message not found in console output")
    }
}


func TestFileOutput(t *testing.T) {
    // Check logging only to file without console output.
    logFile := filepath.Join(os.TempDir(), "log.txt")
    defer os.Remove(logFile)

    config := logger.LogConfig{
        FilePath:      logFile, // File path specified
        Format:        "standard",
        FileLevel:     "debug",
        ConsoleLevel:  "info",
        ConsoleOutput: false,
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    // Log a message and check that it is written to the file
    log.Info("Test informational message to file")

    // Read the file content and check for the message
    data, err := os.ReadFile(logFile)
    if err != nil {
        t.Fatalf("Failed to read log file: %v", err)
    }

    content := string(data)
    if !strings.Contains(content, "Test informational message to file") {
        t.Errorf("Message not found in log file")
    }
}

func TestNoFileLoggingWhenFilePathNotSet(t *testing.T) {
    // Check that log file is not created if file path is not set.
    config := logger.LogConfig{
        FilePath:      "", // File path not specified
        Format:        "standard",
        FileLevel:     "debug",
        ConsoleLevel:  "info",
        ConsoleOutput: false,
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    tempFile := filepath.Join(os.TempDir(), "unused_log.txt")
    defer os.Remove(tempFile)

    // Log a message and check that the file is not created
    log.Info("This message should not appear in any file")

    _, err = os.Stat(tempFile)
    if !os.IsNotExist(err) {
        t.Errorf("Log file should not be created when FilePath is not set")
    }
}


func TestLogRotationWithCompression(t *testing.T) {
    // Check that log files are correctly rotated and compressed.
    logFile := filepath.Join(os.TempDir(), "log_rotation.txt")
    defer os.RemoveAll(filepath.Dir(logFile))

    config := logger.LogConfig{
        FilePath:      logFile,
        Format:        "standard",
        FileLevel:     "debug",
        ConsoleLevel:  "info",
        ConsoleOutput: false,
        EnableRotation: true,
        RotationConfig: logger.RotationConfig{
            MaxSize:    1,  // 1 MB
            MaxBackups: 2,
            MaxAge:     1,  // 1 day
            Compress:   true,
        },
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    // Write enough messages to check rotation and compression
    smallMessage := strings.Repeat("A", 1024*10) // 10 KB
    for i := 0; i < 110; i++ {
        log.Info("Message number", i, smallMessage)
    }

    // Wait for rotation to occur
    time.Sleep(2 * time.Second)

    // Check that rotation and compression occurred
    files, err := os.ReadDir(filepath.Dir(logFile))
    if err != nil {
        t.Fatalf("Failed to read log directory: %v", err)
    }

    compressedFiles := 0
    for _, file := range files {
        if strings.HasSuffix(file.Name(), ".gz") {
            compressedFiles++
        }
    }

    if compressedFiles == 0 {
        t.Errorf("No compressed files found")
    }
}


func TestLogRotationWithoutCompression(t *testing.T) {
    // Check log rotation without compression.
    logFile := filepath.Join(os.TempDir(), "log_rotation.txt")
    defer os.RemoveAll(filepath.Dir(logFile))

    config := logger.LogConfig{
        FilePath:      logFile,
        Format:        "standard",
        FileLevel:     "debug",
        ConsoleLevel:  "info",
        ConsoleOutput: false,
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

    // Write enough messages to trigger log rotation
    smallMessage := strings.Repeat("A", 1024*10) // 10 KB
    for i := 0; i < 110; i++ {                   // 110 * 10 KB = 1.1 MB
        log.Info("Message number", i, smallMessage)
    }

    // Wait for rotation to occur
    time.Sleep(1 * time.Second)

    // Check the number of log files after rotation
    files, err := os.ReadDir(filepath.Dir(logFile))
    if err != nil {
        t.Fatalf("Failed to read log directory: %v", err)
    }

    logFiles := 0
    for _, file := range files {
        if file.Name() == filepath.Base(logFile) || strings.HasPrefix(file.Name(), strings.TrimSuffix(filepath.Base(logFile), ".txt")) {
            logFiles++
            t.Logf("Found file: %s", file.Name())
        }
    }

    if logFiles < 2 {
        t.Errorf("Log rotation did not occur as expected, found %d files", logFiles)
    }
}

func TestDefaultConfig(t *testing.T) {
    // Check default values when creating logger.
    config := logger.LogConfig{}

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    // Check default values
    if log.Config.FilePath != "" {
        t.Errorf("Expected default FilePath to be empty, got '%s'", log.Config.FilePath)
    }
    if log.Config.Format != "standard" {
        t.Errorf("Expected default Format to be 'standard', got '%s'", log.Config.Format)
    }
    if log.Config.FileLevel != "warning" {
        t.Errorf("Expected default FileLevel to be 'warning', got '%s'", log.Config.FileLevel)
    }
    if log.Config.ConsoleLevel != "warning" {
        t.Errorf("Expected default ConsoleLevel to be 'warning', got '%s'", log.Config.ConsoleLevel)
    }
    if log.Config.RotationConfig.MaxSize != 10 {
        t.Errorf("Expected default RotationConfig.MaxSize to be 10, got %d", log.Config.RotationConfig.MaxSize)
    }
    if log.Config.RotationConfig.MaxBackups != 7 {
        t.Errorf("Expected default RotationConfig.MaxBackups to be 7, got %d", log.Config.RotationConfig.MaxBackups)
    }
    if log.Config.RotationConfig.MaxAge != 30 {
        t.Errorf("Expected default RotationConfig.MaxAge to be 30, got %d", log.Config.RotationConfig.MaxAge)
    }
}


func TestLogMethods(t *testing.T) {
    // Check that logging methods work correctly and output to console.
    var consoleOutput bytes.Buffer

    // Save original os.Stdout to restore later
    originalStdout := os.Stdout

    // Create a pipe to redirect stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    config := logger.LogConfig{
        FilePath:      "", // File path not specified, logging will be to console only
        Format:        "standard",
        FileLevel:     "trace",
        ConsoleLevel:  "trace",
        ConsoleOutput: true,
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

    // Log messages to check all logging methods
    log.Trace("TRACE level message")
    log.Debug("Debug message")
    log.Info("Informational message")
    log.Warning("Warning")
    log.Error("Error message")

    log.Tracef("TRACE level message: %d", 1)
    log.Debugf("Debug message: %d", 2)
    log.Infof("Informational message: %d", 3)
    log.Warningf("Warning message: %d", 4)
    log.Errorf("Error message: %d", 5)

    log.Traceln("TRACE level message with newline")
    log.Debugln("Debug message with newline")
    log.Infoln("Informational message with newline")
    log.Warningln("Warning message with newline")
    log.Errorln("Error message with newline")

    // Close the writer to finish the goroutine
    w.Close()
    <-done

    // Restore original os.Stdout
    os.Stdout = originalStdout

    // Check console output
    output := consoleOutput.String()
    lines := strings.Split(output, "\n")

    pidStr := fmt.Sprintf("PID: %d", os.Getpid())

    for _, line := range lines {
        if line == "" {
            continue
        }
        // Check that the line starts with '['
        if !strings.HasPrefix(line, "[") {
            t.Errorf("Expected log entry to start with '[', got '%s'", line)
        }
        // Check for PID
        if !strings.Contains(line, pidStr) {
            t.Errorf("Expected PID '%s' in log entry, got '%s'", pidStr, line)
        }
        // Check for file path and line number
        if !strings.Contains(line, ".go:") {
            t.Errorf("Expected file path and line number in log entry, got '%s'", line)
        }
    }

    // Check for expected messages
    expectedMessages := []string{
        "TRACE level message",
        "Debug message",
        "Informational message",
        "Warning",
        "Error message",
        "TRACE level message: 1",
        "Debug message: 2",
        "Informational message: 3",
        "Warning message: 4",
        "Error message: 5",
        "TRACE level message with newline",
        "Debug message with newline",
        "Informational message with newline",
        "Warning message with newline",
        "Error message with newline",
    }

    for _, msg := range expectedMessages {
        if !strings.Contains(output, msg) {
            t.Errorf("Expected '%s' in output, got '%s'", msg, output)
        }
    }
}

func TestLogToConsoleOnly(t *testing.T) {
    // Check that logging occurs only to console and not to file.
    var consoleOutput bytes.Buffer
    originalStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    config := logger.LogConfig{
        FilePath:      "", // File path not specified
        Format:        "standard",
        FileLevel:     "info",
        ConsoleLevel:  "info",
        ConsoleOutput: true,
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    done := make(chan bool)
    go func() {
        io.Copy(&consoleOutput, r)
        done <- true
    }()

    // Log a message to check console output
    log.Info("Test message for console only")

    // Finish writing to console
    w.Close()
    <-done
    os.Stdout = originalStdout

    // Check that the message is present in the console
    output := consoleOutput.String()
    if !strings.Contains(output, "Test message for console only") {
        t.Errorf("Expected 'Test message for console only' in console output, got '%s'", output)
    }
}

func TestLogToFileOnly(t *testing.T) {
    // Check that logging occurs only to file and not to console.
    logFile := filepath.Join(os.TempDir(), "test_log_to_file_only.txt")
    defer os.Remove(logFile)

    config := logger.LogConfig{
        FilePath:      logFile, // File path specified
        Format:        "standard",
        FileLevel:     "info",
        ConsoleLevel:  "info",
        ConsoleOutput: false,
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    // Log a message to check file output
    log.Info("Test message for file only")

    // Read the file content and check for the message
    data, err := os.ReadFile(logFile)
    if err != nil {
        t.Fatalf("Failed to read log file: %v", err)
    }

    content := string(data)
    if !strings.Contains(content, "Test message for file only") {
        t.Errorf("Expected 'Test message for file only' in file output")
    }
}

func TestLogToFileAndConsole(t *testing.T) {
    // Check that logging occurs simultaneously to file and console.
    var consoleOutput bytes.Buffer
    originalStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    logFile := filepath.Join(os.TempDir(), "test_log_to_file_and_console.txt")
    defer os.Remove(logFile)

    config := logger.LogConfig{
        FilePath:      logFile, // File path specified
        Format:        "standard",
        FileLevel:     "info",
        ConsoleLevel:  "info",
        ConsoleOutput: true,
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    done := make(chan bool)
    go func() {
        io.Copy(&consoleOutput, r)
        done <- true
    }()

    // Log a message to check output to file and console
    log.Info("Test message for file and console")

    // Finish writing to console
    w.Close()
    <-done
    os.Stdout = originalStdout

    // Check for the message in the console
    consoleOutputStr := consoleOutput.String()
    if !strings.Contains(consoleOutputStr, "Test message for file and console") {
        t.Errorf("Expected 'Test message for file and console' in console output")
    }

    // Check for the message in the file
    fileOutput, err := os.ReadFile(logFile)
    if err != nil {
        t.Fatalf("Failed to read log file: %v", err)
    }

    if !strings.Contains(string(fileOutput), "Test message for file and console") {
        t.Errorf("Expected 'Test message for file and console' in file output")
    }
}

func TestLogInJsonFormat(t *testing.T) {
    // Check that logging occurs in JSON format.
    var consoleOutput bytes.Buffer
    originalStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    logFile := filepath.Join(os.TempDir(), "test_log_json_format.txt")
    defer os.Remove(logFile)

    config := logger.LogConfig{
        FilePath:      logFile, // File path specified
        Format:        "json",  // JSON format specified
        FileLevel:     "info",
        ConsoleLevel:  "info",
        ConsoleOutput: true,
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    done := make(chan bool)
    go func() {
        io.Copy(&consoleOutput, r)
        done <- true
    }()

    // Log a message to check JSON output
    log.Info("Test message in JSON format")

    // Finish writing to console
    w.Close()
    <-done
    os.Stdout = originalStdout

    // Check for JSON formatted message in the console
    consoleOutputStr := consoleOutput.String()
    if !strings.Contains(consoleOutputStr, `"message":"Test message in JSON format"`) {
        t.Errorf("Expected JSON formatted 'Test message in JSON format' in console output")
    }

    // Check for JSON formatted message in the file
    fileOutput, err := os.ReadFile(logFile)
    if err != nil {
        t.Fatalf("Failed to read log file: %v", err)
    }

    if !strings.Contains(string(fileOutput), `"message":"Test message in JSON format"`) {
        t.Errorf("Expected JSON formatted 'Test message in JSON format' in file output")
    }
}

func TestDefaultValues(t *testing.T) {
    // Check that default values are correctly substituted when partial configuration is provided.
    config := logger.LogConfig{
        FilePath: "", // File path not specified
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    // Check default values
    if log.Config.Format != "standard" {
        t.Errorf("Expected default Format to be 'standard', got '%s'", log.Config.Format)
    }
    if log.Config.FileLevel != "warning" {
        t.Errorf("Expected default FileLevel to be 'warning', got '%s'", log.Config.FileLevel)
    }
    if log.Config.ConsoleLevel != "warning" {
        t.Errorf("Expected default ConsoleLevel to be 'warning', got '%s'", log.Config.ConsoleLevel)
    }
    if log.Config.RotationConfig.MaxSize != 10 {
        t.Errorf("Expected default RotationConfig.MaxSize to be 10, got %d", log.Config.RotationConfig.MaxSize)
    }
    if log.Config.RotationConfig.MaxBackups != 7 {
        t.Errorf("Expected default RotationConfig.MaxBackups to be 7, got %d", log.Config.RotationConfig.MaxBackups)
    }
    if log.Config.RotationConfig.MaxAge != 30 {
        t.Errorf("Expected default RotationConfig.MaxAge to be 30, got %d", log.Config.RotationConfig.MaxAge)
    }
}

func TestPrintMethods(t *testing.T) {
    // Check that Print, Printf, and Println methods work correctly.
    var consoleOutput bytes.Buffer
    originalStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    config := logger.LogConfig{
        FilePath:      "", // File path not specified
        Format:        "standard",
        FileLevel:     "info",
        ConsoleLevel:  "info",
        ConsoleOutput: true,
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    done := make(chan bool)
    go func() {
        io.Copy(&consoleOutput, r)
        done <- true
    }()

    // Log messages to check Print, Printf, and Println methods
    log.Print("Test Print message")
    log.Printf("Test %s message", "Printf")
    log.Println("Test Println message")

    // Finish writing to console
    w.Close()
    <-done
    os.Stdout = originalStdout

    // Check console output
    output := consoleOutput.String()
    if !strings.Contains(output, "Test Print message") {
        t.Errorf("Expected 'Test Print message' in console output, got '%s'", output)
    }
    if !strings.Contains(output, "Test Printf message") {
        t.Errorf("Expected 'Test Printf message' in console output, got '%s'", output)
    }
    if !strings.Contains(output, "Test Println message") {
        t.Errorf("Expected 'Test Println message' in console output, got '%s'", output)
    }
}

func TestEnsureLoggerInitializedFunction(t *testing.T) {
    // Check that logger is initialized correctly.
    var consoleOutput bytes.Buffer
    originalStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    done := make(chan bool)
    go func() {
        io.Copy(&consoleOutput, r)
        done <- true
    }()

    // Log a message before explicit initialization
    logger.Info("Test message before explicit initialization")

    // Close the writer and wait for the goroutine to finish
    w.Close()
    <-done
    os.Stdout = originalStdout

    // Check console output
    output := consoleOutput.String()
    if !strings.Contains(output, "Test message before explicit initialization") {
        t.Errorf("Expected 'Test message before explicit initialization' in console output, got '%s'", output)
    }
}