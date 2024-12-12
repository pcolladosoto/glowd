package firefly

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	glowdTypes "github.com/scitags/flowd-go/types"
)

func (b *FireflyBackend) sendFirefly(flowID glowdTypes.FlowID) error {
	dialNet := "udp6"
	if !strings.Contains(flowID.Dst.IP.String(), ":") {
		dialNet = "udp4"
	}
	conn, err := net.Dial(dialNet, fmt.Sprintf("%s:%d", flowID.Dst.IP, b.FireflyDestinationPort))
	if err != nil {
		return fmt.Errorf("couldn't initialize UDP socket: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Warn("error closing UDP socket", "err", err)
		}
	}()

	var localFirefly firefly
	localFirefly.Version = FIREFLY_VERSION
	localFirefly.FlowLifecycle.State = flowID.State.String()
	localFirefly.FlowLifecycle.CurrentTime = time.Now().UTC().Format(TIME_FORMAT)
	localFirefly.FlowID.AFI = fmt.Sprintf("ipv%s", string(dialNet[len(dialNet)-1]))
	localFirefly.FlowID.SrcIP = flowID.Src.IP.String()
	localFirefly.FlowID.DstIP = flowID.Dst.IP.String()
	localFirefly.FlowID.Protocol = flowID.Protocol.String()
	localFirefly.FlowID.SrcPort = flowID.Src.Port
	localFirefly.FlowID.DstPort = flowID.Dst.Port
	localFirefly.Context.ExperimentID = flowID.Experiment
	localFirefly.Context.ActivityID = flowID.Activity
	localFirefly.Context.Application = APPLICATION

	if flowID.State == glowdTypes.START {
		localFirefly.FlowLifecycle.StartTime = flowID.StartTs.Format(TIME_FORMAT)
	} else if flowID.State == glowdTypes.END {
		localFirefly.FlowLifecycle.StartTime = flowID.StartTs.Format(TIME_FORMAT)
		localFirefly.FlowLifecycle.EndTime = flowID.EndTs.Format(TIME_FORMAT)
	} else {
		slog.Warn("got a flow with a wrong state", "flowID.State", flowID.State.String())
	}

	payload, err := json.Marshal(localFirefly)
	if err != nil {
		return fmt.Errorf("error marshalling firefly: %w", err)
	}

	if b.PrependSyslog {
		syslogHeader := []byte(fmt.Sprintf(SYSLOG_HEADER, localFirefly.FlowLifecycle.CurrentTime))
		payload = append(syslogHeader, payload...)
	}

	slog.Debug("sending firefly", "dst", flowID.Dst.IP)
	_, err = conn.Write(payload)
	if err != nil {
		return fmt.Errorf("couldn't send the firefly: %w", err)
	}

	return nil
}
