package logger

// LoggerConfig holds the configuration for the logger.
type LoggerConfig struct {
	// Level is the minimum log level (debug, info, warn, error, fatal, panic).
	Level string `yaml:"level" json:"level"`
	// Format is the log format (json, console).
	Format string `yaml:"format" json:"format"`
	// Output is the output destination (stdout, stderr, file).
	Output string `yaml:"output" json:"output"`
	// OutputPath is the file path when Output is "file".
	OutputPath string `yaml:"output_path" json:"output_path"`
	// DisableCaller disables caller annotation in logs.
	DisableCaller bool `yaml:"disable_caller" json:"disable_caller"`
	// DisableStacktrace disables stack traces for error and above.
	DisableStacktrace bool `yaml:"disable_stacktrace" json:"disable_stacktrace"`
	// Development enables development mode (DPanic instead of Panic).
	Development bool `yaml:"development" json:"development"`
}

// DefaultLoggerConfig returns a LoggerConfig with sensible defaults.
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:             "info",
		Format:            "json",
		Output:            "stdout",
		OutputPath:        "",
		DisableCaller:     false,
		DisableStacktrace: false,
		Development:       false,
	}
}
