package netlink

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/prometheus/procfs"
	glowd "github.com/scitags/flowd-go"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// Note these are not the 'correct' constants, but the BPF_*
// counterparts do have the correct values!
const (
	TCP_ESTABLISHED = unix.BPF_TCP_ESTABLISHED
	TCP_LISTEN      = unix.BPF_TCP_LISTEN
)

var (
	Defaults = map[string]interface{}{
		"pollIntervalSeconds": 5,
		"experimentID":        55,
		"activityID":          55,
	}
)

type fTuple struct {
	SrcIP   string
	SrcPort uint16
	DstIP   string
	DstPort uint16
}

type NetlinkPlugin struct {
	pFS           procfs.FS
	netlinkHandle *netlink.Handle

	PollIntervalSeconds int
	ExperimentID        uint32
	ActivityID          uint32
}

func (p *NetlinkPlugin) String() string {
	return "netlink"
}

func (p *NetlinkPlugin) Init() error {
	slog.Debug("initialising the netlink plugin")

	fs, err := procfs.NewFS("/proc")
	if err != nil {
		return fmt.Errorf("couldn't initialise the procfs filesystem: %w", err)
	}
	p.pFS = fs

	h, err := netlink.NewHandle()
	if err != nil {
		return fmt.Errorf("couldn't get a netlink handle: %w", err)
	}
	p.netlinkHandle = h

	return nil
}

// Be sure to check sock_diag(7) and netlink(7):
//
//	https://www.man7.org/linux/man-pages/man7/sock_diag.7.html
//	https://www.man7.org/linux/man-pages/man7/netlink.7.html
func (p *NetlinkPlugin) getTCPSockets(family uint8) []*netlink.Socket {
	// Note how SocketDiagTCP forces the polled states to 0xFFFF. We could
	// reimplement the method and set the state to TCP_ESTABLISHED to avoid
	// having to do the filtering ourselves. We'll take this option for now
	// though...
	tcpConns, err := p.netlinkHandle.SocketDiagTCP(family)
	if err != nil {
		slog.Warn("error getting tcp4 information", "err", err)
		return nil
	}

	// Only keep established connections. We'll do the filtering in place!
	n := 0
	for _, conn := range tcpConns {
		if conn.State == TCP_ESTABLISHED && !IsIPPrivate(conn.ID.Source) && !IsIPPrivate(conn.ID.Destination) {
			tcpConns[n] = conn
			n++
		}
	}
	tcpConns = tcpConns[:n]

	slog.Debug("polled tcp sockets", "family", family, "n", len(tcpConns))
	for i, conn := range tcpConns {
		slog.Debug("tcp socket conn", "family", family, "i", i, "state", conn.State,
			"sAddr", conn.ID.Source, "sPort", conn.ID.SourcePort,
			"dAddr", conn.ID.Destination, "dPort", conn.ID.DestinationPort)
	}
	return tcpConns
}

func (p *NetlinkPlugin) fTupleToFlowID(f fTuple, conn *netlink.Socket, state glowd.FlowState) glowd.FlowID {
	return glowd.FlowID{
		State:    state,
		StartTs:  time.Now(),
		Protocol: glowd.TCP,
		Src: glowd.IPPort{
			IP:   conn.ID.Source,
			Port: conn.ID.SourcePort,
		},
		Dst: glowd.IPPort{
			IP:   conn.ID.Destination,
			Port: conn.ID.DestinationPort,
		},
		Activity:   p.ActivityID,
		Experiment: p.ExperimentID,
	}
}

func (p *NetlinkPlugin) Run(done <-chan struct{}, outChan chan<- glowd.FlowID) {
	slog.Debug("running the netlink plugin")

	activeIPv4Flows := map[fTuple]struct{}{}
	// activeIPv6Flows := map[glowd.FlowID]struct{}{}
	for {
		select {
		case <-time.Tick(time.Second * time.Duration(p.PollIntervalSeconds)):
			// TODO: cache connections and detect start/end events
			for _, conn := range p.getTCPSockets(unix.AF_INET) {
				tmp := fTuple{
					SrcIP:   conn.ID.Source.String(),
					SrcPort: conn.ID.SourcePort,
					DstIP:   conn.ID.Destination.String(),
					DstPort: conn.ID.DestinationPort,
				}

				_, ok := activeIPv4Flows[tmp]
				if !ok {
					activeIPv4Flows[tmp] = struct{}{}

					outChan <- p.fTupleToFlowID(tmp, conn, glowd.START)

					continue
				}

				delete(activeIPv4Flows, tmp)
			}

			// for id := range activeIPv4Flows {
			// 	outChan <- p.fTupleToFlowID(id, conn, glowd.END)
			// }

			// p.getTCPSockets(unix.AF_INET6)
		case <-done:
			slog.Debug("cleanly exiting the np plugin")
			return
		}
	}
}

func (p *NetlinkPlugin) Cleanup() error {
	slog.Debug("cleaning up the netlink plugin")
	p.netlinkHandle.Close()
	return nil
}
