package main

import (
	"fmt"
	"log/slog"

	glowdTypes "github.com/scitags/flowd-go/types"
)

func initPlugins(plugins []glowdTypes.Plugin) error {
	for _, plugin := range plugins {
		if err := plugin.Init(); err != nil {
			return fmt.Errorf("error setting up plugin %s: %w", plugin, err)
		}
	}
	return nil
}

func initBackends(backends []glowdTypes.Backend) error {
	for _, backend := range backends {
		if err := backend.Init(); err != nil {
			return fmt.Errorf("error setting up backend %s: %w", backend, err)
		}
	}
	return nil
}

func cleanupPlugins(plugins []glowdTypes.Plugin) {
	for _, plugin := range plugins {
		if err := plugin.Cleanup(); err != nil {
			slog.Error("error cleaning up plugin", "plugin", plugin, "err", err)
		}
	}
}

func cleanupBackends(backends []glowdTypes.Backend) {
	for _, backend := range backends {
		if err := backend.Cleanup(); err != nil {
			slog.Error("error cleaning up backend", "backend", backend, "err", err)
		}
	}
}
