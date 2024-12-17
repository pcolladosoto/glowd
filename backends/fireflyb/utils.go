package fireflyb

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/scitags/flowd-go/netlink"
	glowdTypes "github.com/scitags/flowd-go/types"
)

func (b *FireflyBackend) sendFirefly(flowID glowdTypes.FlowID) error {
	dialNet := "udp6"
	if flowID.Family == glowdTypes.IPv4 {
		dialNet = "udp4"
	}

	var err error
	conn, err := net.Dial(dialNet, fmt.Sprintf("%s:%d", flowID.Dst.IP, b.FireflyDestinationPort))
	if err != nil {
		return fmt.Errorf("couldn't initialize UDP socket: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Warn("error closing UDP socket", "err", err)
		}
	}()

	var nlInfo *netlink.InetDiagTCPInfoResp
	if b.AddNetlinkContext {
		nlInfo, err = b.addNetlinkContext(uint8(flowID.Family), flowID.Src.Port, flowID.Dst.Port)
		if err != nil {
			slog.Warn("error getting info from netlink, proceeding without it", "err", err)

			// Even though this should already be the case, be explicit!
			nlInfo = nil
		}
	}

	var localFirefly glowdTypes.Firefly
	localFirefly.Version = glowdTypes.FIREFLY_VERSION

	localFirefly.FlowLifecycle.State = flowID.State.String()
	localFirefly.FlowLifecycle.CurrentTime = time.Now().UTC().Format(glowdTypes.TIME_FORMAT)

	localFirefly.FlowID.AFI = flowID.Family.String()
	localFirefly.FlowID.SrcIP = flowID.Src.IP.String()
	localFirefly.FlowID.DstIP = flowID.Dst.IP.String()
	localFirefly.FlowID.Protocol = flowID.Protocol.String()
	localFirefly.FlowID.SrcPort = flowID.Src.Port
	localFirefly.FlowID.DstPort = flowID.Dst.Port

	localFirefly.Context.ExperimentID = flowID.Experiment
	localFirefly.Context.ActivityID = flowID.Activity
	localFirefly.Context.Application = glowdTypes.APPLICATION

	localFirefly.Netlink = nlInfo

	localFirefly.ParseTimeStamps(flowID)

	// TODO: If src IP address is private, get one through STUN!

	payload, err := json.Marshal(localFirefly)
	if err != nil {
		return fmt.Errorf("error marshalling firefly: %w", err)
	}

	if b.PrependSyslog {
		syslogHeader := []byte(fmt.Sprintf(glowdTypes.SYSLOG_HEADER, localFirefly.FlowLifecycle.CurrentTime))
		payload = append(syslogHeader, payload...)
	}

	slog.Debug("sending firefly", "dst", flowID.Dst.IP)
	_, err = conn.Write(payload)
	if err != nil {
		return fmt.Errorf("couldn't send the firefly: %w", err)
	}

	return nil
}

func (b *FireflyBackend) addNetlinkContext(family uint8, srcPort, dstPort uint16) (*netlink.InetDiagTCPInfoResp, error) {
	nlReplies, err := netlink.NewTCPDiagRequest(family, srcPort, dstPort).ExecuteRequest()
	if err != nil {
		return nil, fmt.Errorf("couldn't execute the netlink request: %w", err)
	}

	switch len(nlReplies) {
	case 0:
		return nil, fmt.Errorf("got no information from netlink...")
	case 1:
		return nlReplies[0], nil
	default:
		slog.Warn("got information for more than one flow...")

		// TODO: Filter replies from netlink based on IPv{4,6} addresses
		return nlReplies[0], nil
	}
}
