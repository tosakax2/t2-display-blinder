package config

import (
	"time"
)

// Config represents the application configuration settings.
type Config struct {
	// ScreenOffDelay is the duration to wait before turning off the screen.
	ScreenOffDelay time.Duration
	// BlinderDismissOnMouseMove determines whether moving the mouse dismisses the blinder.
	BlinderDismissOnMouseMove bool
	// WindowWidth is the default width of the control window.
	WindowWidth int
	// WindowHeight is the default height of the control window.
	WindowHeight int
}

// Default returns the default configuration.
func Default() *Config {
	return &Config{
		ScreenOffDelay:            1 * time.Second,
		BlinderDismissOnMouseMove: true,
		WindowWidth:               450,
		WindowHeight:              580,
	}
}
