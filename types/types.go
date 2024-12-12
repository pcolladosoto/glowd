package types

import (
	"net"
	"strings"
	"time"
)

type FlowID struct {
	State      FlowState
	Protocol   Protocol
	Src        IPPort
	Dst        IPPort
	Experiment uint32
	Activity   uint32
	StartTs    time.Time
	EndTs      time.Time
	NetLink    string
}

type IPPort struct {
	IP   net.IP
	Port uint16
}

type Protocol int

const (
	TCP Protocol = iota
	UDP
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
)

func (p Protocol) String() string {
	return locotorpMap[p]
}

func ParseProtocol(proto string) (Protocol, bool) {
	p, ok := protocolMap[strings.ToUpper(proto)]
	return p, ok
}

type FlowState int

const (
	START FlowState = iota
	END
)

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
