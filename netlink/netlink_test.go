package netlink

import (
	"encoding/json"
	"fmt"
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
	resps, err := NewTCPDiagRequest(unix.AF_INET, 0, 0).ExecuteRequest()
	if err != nil {
		t.Errorf("failed to execute request: %v", err)
	}

	for i, resp := range resps {
		mResp, err := json.MarshalIndent(resp, "", "    ")
		if err != nil {
			t.Errorf("failed to marshall response %d: %v", i, err)
		}
		fmt.Printf("response %d:\n%s\n", i, mResp)
	}
}
