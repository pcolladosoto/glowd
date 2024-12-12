package netlink

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func init() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Remove time.
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}
			// Remove the directory from the source's filename.
			if a.Key == slog.SourceKey {
				source := a.Value.Any().(*slog.Source)
				source.File = filepath.Base(source.File)
			}
			return a
		},
	}))
	slog.SetDefault(logger)
}

func TestRequestExecution(t *testing.T) {
	plugin := NetlinkPlugin{}
	if err := plugin.Init(); err != nil {
		t.Errorf("couldn't initialise the netlink plugin: %v", err)
	}

	req := plugin.MakeRequest(unix.AF_INET, 0, 0)
	if err := plugin.ExecuteRequest(req); err != nil {
		t.Errorf("failed to execute request: %v", err)
	}
}
