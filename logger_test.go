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
    // Проверяем логирование только в консоль, без записи в файл.
    var consoleOutput bytes.Buffer
    originalStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    config := logger.LogConfig{
        FilePath:      "", // Не указан путь к файлу
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

    // Логируем сообщение и проверяем, что оно появится в консоли
    log.Info("Test informational message")

    // Завершаем запись в консоль
    w.Close()
    <-done
    os.Stdout = originalStdout

    // Проверяем вывод в консоль
    output := consoleOutput.String()
    if !strings.Contains(output, "Test informational message") {
        t.Errorf("Message not found in console output")
    }
}


func TestFileOutput(t *testing.T) {
    // Проверяем логирование только в файл без вывода в консоль.
    logFile := filepath.Join(os.TempDir(), "log.txt")
    defer os.Remove(logFile)

    config := logger.LogConfig{
        FilePath:      logFile, // Указан путь к файлу
        Format:        "standard",
        FileLevel:     "debug",
        ConsoleLevel:  "info",
        ConsoleOutput: false,
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    // Логируем сообщение и проверяем, что оно записывается в файл
    log.Info("Test informational message to file")

    // Считываем содержимое файла и проверяем наличие сообщения
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
    // Проверяем, что лог-файл не создается, если путь к файлу не задан.
    config := logger.LogConfig{
        FilePath:      "", // Путь к файлу не указан
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

    // Логируем сообщение, проверяем, что файл не создается
    log.Info("This message should not appear in any file")

    _, err = os.Stat(tempFile)
    if !os.IsNotExist(err) {
        t.Errorf("Log file should not be created when FilePath is not set")
    }
}


func TestLogRotationWithCompression(t *testing.T) {
    // Проверяем, что лог-файлы корректно ротируются и сжимаются.
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

    // Пишем достаточное количество сообщений для проверки ротации и сжатия
    smallMessage := strings.Repeat("A", 1024*10) // 10 KB
    for i := 0; i < 110; i++ {
        log.Info("Message number", i, smallMessage)
    }

    // Ждем, чтобы ротация произошла
    time.Sleep(2 * time.Second)

    // Проверяем, что ротация и сжатие файлов произошли
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
    // Проверяем ротацию логов без сжатия.
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

    // Пишем достаточное количество сообщений для ротации логов
    smallMessage := strings.Repeat("A", 1024*10) // 10 KB
    for i := 0; i < 110; i++ {                   // 110 * 10 KB = 1.1 MB
        log.Info("Message number", i, smallMessage)
    }

    // Ждем, чтобы ротация произошла
    time.Sleep(1 * time.Second)

    // Проверяем количество лог-файлов после ротации
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

// fileInfoSize returns the size of a file in bytes.
// func fileInfoSize(dir, name string) int64 {
//     info, err := os.Stat(filepath.Join(dir, name))
//     if err != nil {
//         return 0
//     }
//     return info.Size()
// }

func TestDefaultConfig(t *testing.T) {
    // Проверяем подстановку значений по умолчанию при создании логгера.
    config := logger.LogConfig{}

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    // Проверяем значения по умолчанию
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
    // Проверяем, что методы логирования работают корректно и выводятся в консоль.
    var consoleOutput bytes.Buffer

    // Сохраняем оригинальный os.Stdout для восстановления позже
    originalStdout := os.Stdout

    // Создаем пайп для перенаправления stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    config := logger.LogConfig{
        FilePath:      "", // Путь к файлу не указан, логирование будет только в консоль
        Format:        "standard",
        FileLevel:     "trace",
        ConsoleLevel:  "trace",
        ConsoleOutput: true,
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    // Запускаем горутину для чтения из пайпа
    done := make(chan bool)
    go func() {
        io.Copy(&consoleOutput, r)
        done <- true
    }()

    // Логируем сообщения для проверки всех методов логирования
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

    // Закрываем writer для завершения горутины
    w.Close()
    <-done

    // Восстанавливаем оригинальный os.Stdout
    os.Stdout = originalStdout

    // Проверяем вывод в консоль
    output := consoleOutput.String()
    lines := strings.Split(output, "\n")

    pidStr := fmt.Sprintf("PID: %d", os.Getpid())

    for _, line := range lines {
        if line == "" {
            continue
        }
        // Проверяем, что строка начинается с '['
        if !strings.HasPrefix(line, "[") {
            t.Errorf("Expected log entry to start with '[', got '%s'", line)
        }
        // Проверяем наличие PID
        if !strings.Contains(line, pidStr) {
            t.Errorf("Expected PID '%s' in log entry, got '%s'", pidStr, line)
        }
        // Проверяем наличие пути к файлу и номера строки
        if !strings.Contains(line, ".go:") {
            t.Errorf("Expected file path and line number in log entry, got '%s'", line)
        }
    }

    // Проверяем наличие ожидаемых сообщений
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
    // Проверяем, что логирование происходит только в консоль и не записывается в файл.
    var consoleOutput bytes.Buffer
    originalStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    config := logger.LogConfig{
        FilePath:      "", // Путь к файлу не указан
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

    // Логируем сообщение для проверки вывода в консоль
    log.Info("Test message for console only")

    // Завершаем запись в консоль
    w.Close()
    <-done
    os.Stdout = originalStdout

    // Проверяем, что сообщение присутствует в консоли
    output := consoleOutput.String()
    if !strings.Contains(output, "Test message for console only") {
        t.Errorf("Expected 'Test message for console only' in console output, got '%s'", output)
    }
}

func TestLogToFileOnly(t *testing.T) {
    // Проверяем, что логирование происходит только в файл и не выводится в консоль.
    logFile := filepath.Join(os.TempDir(), "test_log_to_file_only.txt")
    defer os.Remove(logFile)

    config := logger.LogConfig{
        FilePath:      logFile, // Указан путь к файлу
        Format:        "standard",
        FileLevel:     "info",
        ConsoleLevel:  "info",
        ConsoleOutput: false,
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    // Логируем сообщение для проверки записи в файл
    log.Info("Test message for file only")

    // Считываем содержимое файла и проверяем наличие сообщения
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
    // Проверяем, что логирование происходит одновременно в файл и в консоль.
    var consoleOutput bytes.Buffer
    originalStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    logFile := filepath.Join(os.TempDir(), "test_log_to_file_and_console.txt")
    defer os.Remove(logFile)

    config := logger.LogConfig{
        FilePath:      logFile, // Указан путь к файлу
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

    // Логируем сообщение для проверки вывода в файл и консоль
    log.Info("Test message for file and console")

    // Завершаем запись в консоль
    w.Close()
    <-done
    os.Stdout = originalStdout

    // Проверяем наличие сообщения в консоли
    consoleOutputStr := consoleOutput.String()
    if !strings.Contains(consoleOutputStr, "Test message for file and console") {
        t.Errorf("Expected 'Test message for file and console' in console output")
    }

    // Проверяем наличие сообщения в файле
    fileOutput, err := os.ReadFile(logFile)
    if err != nil {
        t.Fatalf("Failed to read log file: %v", err)
    }

    if !strings.Contains(string(fileOutput), "Test message for file and console") {
        t.Errorf("Expected 'Test message for file and console' in file output")
    }
}

func TestLogInJsonFormat(t *testing.T) {
    // Проверяем, что логирование происходит в формате JSON.
    var consoleOutput bytes.Buffer
    originalStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    logFile := filepath.Join(os.TempDir(), "test_log_json_format.txt")
    defer os.Remove(logFile)

    config := logger.LogConfig{
        FilePath:      logFile, // Указан путь к файлу
        Format:        "json",  // Указан формат JSON
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

    // Логируем сообщение для проверки JSON-вывода
    log.Info("Test message in JSON format")

    // Завершаем запись в консоль
    w.Close()
    <-done
    os.Stdout = originalStdout

    // Проверяем наличие сообщения в формате JSON в консоли
    consoleOutputStr := consoleOutput.String()
    if !strings.Contains(consoleOutputStr, `"message":"Test message in JSON format"`) {
        t.Errorf("Expected JSON formatted 'Test message in JSON format' in console output")
    }

    // Проверяем наличие сообщения в формате JSON в файле
    fileOutput, err := os.ReadFile(logFile)
    if err != nil {
        t.Fatalf("Failed to read log file: %v", err)
    }

    if !strings.Contains(string(fileOutput), `"message":"Test message in JSON format"`) {
        t.Errorf("Expected JSON formatted 'Test message in JSON format' in file output")
    }
}

func TestDefaultValues(t *testing.T) {
    // Проверяем, что при частичном задании конфигурации корректно подставляются значения по умолчанию.
    config := logger.LogConfig{
        FilePath: "", // Путь к файлу не указан
    }

    log, err := logger.NewLogger(config)
    if err != nil {
        t.Fatalf("Failed to create logger: %v", err)
    }

    // Проверяем значения по умолчанию
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