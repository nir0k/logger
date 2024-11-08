# Changelog

All significant changes to this project will be documented in this file.

## [1.2.0] - 2024-11-09

### Added

- Default values for `LogConfig` fields if they are not provided:
  - `Directory` defaults to `./logs`.
  - `Format` defaults to `standard`.
  - `FileLevel` defaults to `info`.
  - `ConsoleLevel` defaults to `info`.
  - `RotationConfig.MaxSize` defaults to `100` MB.
  - `RotationConfig.MaxBackups` defaults to `7`.
  - `RotationConfig.MaxAge` defaults to `30` days.
- New test `TestDefaultConfig` to verify that default values are correctly set when not provided in the configuration.

## [1.1.0] - 2024-11-08

### Added

- Support for various logging levels for file writing and console output.
- New configuration fields `FileLevel` and `ConsoleLevel` to set the minimum logging level for file and console separately.
- Updated tests to check new features:
  - Separate logging levels for file and console.
  - Console output verification considering the new logging level.
  - File writing verification considering the new logging level.

## [1.0.0] - 2024-11-07

### Added

- Initial release of the logging package for Go.
- Support for multiple logging levels: `Trace`, `Debug`, `Info`, `Warning`, `Error`, `Fatal`.
- Customizable output formats: `standard` and `json`.
- Console output with color-coded logging levels.
- File logging with optional log rotation.
- Log rotation configuration:
  - `MaxSize`: Maximum file size in megabytes before rotation.
  - `MaxBackups`: Maximum number of old log files to keep.
  - `MaxAge`: Maximum number of days to keep old log files.
  - `Compress`: Option to compress rotated log files.
- Detailed tests to verify:
  - Console output.
  - File writing.
  - Log rotation.
  - Compression of rotated log files.
- Added comments and documentation in English in the source code.
- Created `README.md` and `CHANGELOG.md` files with detailed information and instructions.

---