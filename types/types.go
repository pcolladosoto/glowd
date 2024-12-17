package types

import (
	"net"
	"strings"
	"time"

	"github.com/scitags/flowd-go/netlink"
	"golang.org/x/sys/unix"
)

type FlowID struct {
	State      FlowState
	Protocol   Protocol
	Family     Family
	Src        IPPort
	Dst        IPPort
	Experiment uint32
	Activity   uint32
	StartTs    time.Time
	EndTs      time.Time
	CurrentTs  time.Time
	NetLink    netlink.InetDiagTCPInfoResp
}

type IPPort struct {
	IP   net.IP
	Port uint16
}

type Protocol int

type Family int

type FlowState int

const (
	TCP Protocol = iota
	UDP

	START FlowState = iota
	END

	IPv4 Family = unix.AF_INET
	IPv6 Family = unix.AF_INET6
)

// TODO: Find a way to make this nicer. Maybe init() functions is the
// TODO: way to go?
var (
	protocolMap = map[string]Protocol{
		"TCP": TCP,
		"UDP": UDP,
	}

	locotorpMap = map[Protocol]string{
		TCP: "TCP",
		UDP: "UDP",
	}

	flowMap = map[string]FlowState{
		"START": START,
		"END":   END,
	}

	wolfMap = map[FlowState]string{
		START: "START",
		END:   "END",
	}

	familyMap = map[string]Family{
		"IPV4": IPv4,
		"IPV6": IPv4,
	}

	ylimafMap = map[Family]string{
		IPv4: "ipv4",
		IPv6: "ipv6",
	}
)

func (p Protocol) String() string {
	return locotorpMap[p]
}

func ParseProtocol(proto string) (Protocol, bool) {
	p, ok := protocolMap[strings.ToUpper(proto)]
	return p, ok
}

func (f Family) String() string {
	return ylimafMap[f]
}

func ParseFamily(proto string) (Family, bool) {
	f, ok := familyMap[strings.ToUpper(proto)]
	return f, ok
}

func (fs FlowState) String() string {
	return wolfMap[fs]
}

func ParseFlowState(flowState string) (FlowState, bool) {
	fs, ok := flowMap[strings.ToUpper(flowState)]
	return fs, ok
}

type Backend interface {
	Init() error
	Run(<-chan struct{}, <-chan FlowID)
	Cleanup() error
	String() string
}

type Plugin interface {
	Init() error
	Run(<-chan struct{}, chan<- FlowID)
	Cleanup() error
	String() string
}
