package types

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/scitags/flowd-go/netlink"
)

const (
	FIREFLY_VERSION int    = 1
	APPLICATION     string = "flowd-go v1.0.0"

	// Let's replicate 2024-11-02T16:07:01.769470+00:00.
	TIME_FORMAT string = "2006-01-02T15:04:05.999999-07:00"

	SYSLOG_FACILITY_LOCAL0        int    = 16
	SYSLOG_SEVERITY_INFORMATIONAL int    = 6
	SYSLOG_PRIORITY               int    = (SYSLOG_FACILITY_LOCAL0 << 3) | SYSLOG_SEVERITY_INFORMATIONAL
	SYSLOG_VERSION                int    = 1
	SYSLOG_APP_NAME               string = "flowd-go"
	SYSLOG_PROC_ID                string = "-"
	SYSLOG_MSG_ID                 string = "firefly-json"
	SYSLOG_STRUCT_DATA            string = "-"
)

var (
	SYSLOG_HEADER string
)

func init() {
	hostName, err := os.Hostname()
	if err != nil {
		hostName = "no idea!"
	}

	SYSLOG_HEADER = fmt.Sprintf("<%d>%d %%s %s %s %s %s %s ",
		SYSLOG_PRIORITY, SYSLOG_VERSION, hostName,
		SYSLOG_APP_NAME, SYSLOG_PROC_ID, SYSLOG_MSG_ID,
		SYSLOG_STRUCT_DATA,
	)
}

type Firefly struct {
	Version       int `json:"version"`
	FlowLifecycle struct {
		State       string `json:"state"`
		CurrentTime string `json:"current-time"`
		StartTime   string `json:"start-time"`
		EndTime     string `json:"end-time,omitempty"`
	} `json:"flow-lifecycle"`
	FlowID struct {
		AFI      string `json:"afi"`
		SrcIP    string `json:"src-ip"`
		DstIP    string `json:"dst-ip"`
		Protocol string `json:"protocol"`
		SrcPort  uint16 `json:"src-port"`
		DstPort  uint16 `json:"dst-port"`
	} `json:"flow-id"`
	Context struct {
		ExperimentID uint32 `json:"experiment-id"`
		ActivityID   uint32 `json:"activity-id"`
		Application  string `json:"application"`
	} `json:"context"`
	Netlink *netlink.InetDiagTCPInfoResp `json:"netlink,omitempty"`
}

type AuxFirefly struct {
	FlowID FlowID `json:"flow-id"`
}

func (f *AuxFirefly) UnmarshalJSON(in []byte) error {
	jsonIndex := strings.Index(string(in), "{")
	if jsonIndex == -1 {
		return fmt.Errorf("couldn't find the JSON start token '{'...")
	}

	rawFirefly := Firefly{}
	if err := json.Unmarshal(in[jsonIndex:], &rawFirefly); err != nil {
		return fmt.Errorf("error unmarshaling the raw firefly: %w", err)
	}

	flowState, ok := ParseFlowState(rawFirefly.FlowLifecycle.State)
	if !ok {
		return fmt.Errorf("wrong state %s specified", rawFirefly.FlowLifecycle.State)
	}
	family, ok := ParseFamily(rawFirefly.FlowID.AFI)
	if !ok {
		return fmt.Errorf("wrong family %s", rawFirefly.FlowID.AFI)
	}
	protocol, ok := ParseProtocol(rawFirefly.FlowID.Protocol)
	if !ok {
		fmt.Errorf("wrong protocol %s specified", rawFirefly.FlowID.Protocol)
	}

	// Checking IP address parsing can be problematic given both families leverage
	// a 16-bit underlying representation... Check 0:
	// 0: https://stackoverflow.com/questions/22751035/golang-distinguish-ipv4-ipv6
	parseIP := func(rawIP string) (net.IP, error) {
		ipvX := net.ParseIP(rawIP)

		if ipvX == nil {
			return nil, fmt.Errorf("address %s is not a valid IPv{4,6}", rawIP)
		}
		if family == IPv4 && strings.Contains(rawIP, ":") {
			return nil, fmt.Errorf("IPv4 address expected, but it looks like an IPv6...")
		}
		if family == IPv6 && !strings.Contains(rawIP, ":") {
			return nil, fmt.Errorf("IPv6 address expected, but it looks like an IPv4...")
		}

		return ipvX, nil
	}

	srcIP, err := parseIP(rawFirefly.FlowID.SrcIP)
	if err != nil {
		return fmt.Errorf("error parsing source IP: %w", err)
	}
	dstIP, err := parseIP(rawFirefly.FlowID.DstIP)
	if err != nil {
		return fmt.Errorf("error parsing destination IP: %w", err)
	}

	parseTs := func(rawTs string) (time.Time, error) {
		if rawTs == "" {
			return time.Time{}, nil
		}
		return time.Parse(TIME_FORMAT, rawTs)
	}

	startTs, err := parseTs(rawFirefly.FlowLifecycle.StartTime)
	if err != nil {
		return fmt.Errorf("couldn't parse start timestamp: %w", err)
	}
	endTs, err := parseTs(rawFirefly.FlowLifecycle.EndTime)
	if err != nil {
		return fmt.Errorf("couldn't parse end timestamp: %w", err)
	}

	f.FlowID = FlowID{
		State:      flowState,
		Family:     family,
		Protocol:   protocol,
		Src:        IPPort{srcIP, rawFirefly.FlowID.SrcPort},
		Dst:        IPPort{dstIP, rawFirefly.FlowID.DstPort},
		Experiment: rawFirefly.Context.ExperimentID,
		Activity:   rawFirefly.Context.ActivityID,
		StartTs:    startTs,
		EndTs:      endTs,
	}

	return nil
}

// func (f *AuxFirefly) MarshalJSON() ([]byte, error) { return nil, nil }

func (f *Firefly) ParseTimeStamps(flowID FlowID) {
	if !flowID.StartTs.IsZero() {
		f.FlowLifecycle.StartTime = flowID.StartTs.Format(TIME_FORMAT)
	}

	if !flowID.EndTs.IsZero() {
		f.FlowLifecycle.EndTime = flowID.EndTs.Format(TIME_FORMAT)
	}

	f.FlowLifecycle.CurrentTime = time.Now().Format(TIME_FORMAT)
}
