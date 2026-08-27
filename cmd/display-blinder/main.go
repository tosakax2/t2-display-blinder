package main

import (
	"fmt"
	"os"

	"t2-display-blinder/internal/app"
	"t2-display-blinder/internal/config"
)

func main() {
	cfg := config.Default()
	application := app.New(cfg)

	if err := application.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
		os.Exit(1)
	}
}
