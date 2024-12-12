package firefly

import (
	"log/slog"

	glowdTypes "github.com/scitags/flowd-go/types"
)

var (
	Defaults = map[string]interface{}{
		"fireflyDestinationPort": 10514,
		"prependSyslog":          false,
	}
)

type FireflyBackend struct {
	FireflyDestinationPort uint16 `json:"fireflyDestinationPort"`
	PrependSyslog          bool   `json:"prependSyslog"`
}

func (b *FireflyBackend) String() string {
	return "Firefly"
}

// Just implement the glowd.Backend interface
func (b *FireflyBackend) Init() error {
	slog.Debug("initialising the firefly backend")
	return nil
}

func (b *FireflyBackend) Run(done <-chan struct{}, inChan <-chan glowdTypes.FlowID) {
	slog.Debug("running the firefly backend")
	for {
		select {
		case flowID, ok := <-inChan:
			if !ok {
				slog.Warn("somebody closed the input channel!")
				return
			}
			slog.Debug("got a flowID")
			if err := b.sendFirefly(flowID); err != nil {
				slog.Error("error sending the firefly", "err", err)
			}
		case <-done:
			slog.Debug("cleanly exiting the firefly backend")
			return
		}
	}
}

func (b *FireflyBackend) Cleanup() error {
	slog.Debug("cleaning up the firefly backend")
	return nil
}
