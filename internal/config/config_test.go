package config_test

import (
	"testing"
	"time"

	"t2-display-blinder/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.Default()

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	if cfg.ScreenOffDelay != 1*time.Second {
		t.Errorf("expected ScreenOffDelay to be 1s, got %v", cfg.ScreenOffDelay)
	}

	if !cfg.BlinderDismissOnMouseMove {
		t.Errorf("expected BlinderDismissOnMouseMove to be true")
	}

	if cfg.WindowWidth <= 0 || cfg.WindowHeight <= 0 {
		t.Errorf("invalid window dimensions: %dx%d", cfg.WindowWidth, cfg.WindowHeight)
	}
}
