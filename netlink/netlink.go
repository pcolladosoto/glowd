package netlink

import (
	"fmt"
	"log/slog"
	"syscall"
	"time"

	"github.com/prometheus/procfs"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
	"golang.org/x/sys/unix"

	glowdTypes "github.com/scitags/flowd-go/types"
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
		if conn.State == uint8(TCP_ESTABLISHED) && !IsIPPrivate(conn.ID.Source) && !IsIPPrivate(conn.ID.Destination) {
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

// TODO - Figure out why we aren't seeing INET_DIAG_DCTCPINFO or INET_DIAG_BBRINFO messages.
// Plundered from github.com/m-lab/tcp-info. Please note the main logic driving TCP diag
// requests on netlink can be found on [0], which in turn calls into inet_diag_dump_icsk [1].
// Port numbers set to 0 will not be applied as a filter. That is, if you desire to retrieve all
// the sockets simply pass 0, 0 to both the source and destination ports. Note that even if
// filtering on IP addresses is not available, IP information is contained in responses. Not
// only that, from a conceptual point of view at the L4 level ports are univocal IDs in the
// sense that the {src,dst}Port tuple will be unique no matter what the actual IPs are...
//
// 0: https://elixir.bootlin.com/linux/v6.12.4/source/net/ipv4/tcp_diag.c#L181
//
// 1: https://elixir.bootlin.com/linux/v6.12.4/source/net/ipv4/inet_diag.c#L1019
func (p *NetlinkPlugin) MakeRequest(family uint8, srcPort uint16, dstPort uint16) *nl.NetlinkRequest {
	// The entry point to RTNetlink for NLM_F_DUMP requests seems to be [0]. The thing is there
	// are myriad callbacks going around so keeping track of this stuff can get tricky...
	// 0: https://elixir.bootlin.com/linux/v6.12.4/source/net/core/rtnetlink.c#L6597
	req := nl.NewNetlinkRequest(nl.SOCK_DIAG_BY_FAMILY, unix.NLM_F_DUMP|unix.NLM_F_REQUEST)

	// When reading netlink(7) and sock_diag(7) it might look like the below behaves differently than the above,
	// but the 'filtering' is controlled by whether there are non-nil fields in SockDiagReq.ID or not... Checking
	// the references to NLM_F_MATCH [0] there are none outside the header files, so it looks like it won't have
	// much of an effect...
	// 0: https://elixir.bootlin.com/linux/v6.12.4/A/ident/NLM_F_MATCH
	// req := nl.NewNetlinkRequest(nl.SOCK_DIAG_BY_FAMILY, unix.NLM_F_MATCH|unix.NLM_F_REQUEST)

	msg := SockDiagReq{
		Family:   family,
		Protocol: unix.IPPROTO_TCP,
		// tcp-info disregards some connections... Netlink on the other hand doesn't! Be sure to take a look
		// at [0] for an insight into how link states are taken into account when deciding whether to
		// consider sockets or not:
		// 0: https://elixir.bootlin.com/linux/v6.12.4/source/net/ipv4/inet_diag.c#L1019
		// States: TCP_ALL_FLAGS & ^((1 << uint(TCP_SYN_RECV)) |
		// 	(1 << uint(TCP_TIME_WAIT)) |
		// 	(1 << uint(TCP_CLOSE)) |
		// 	(1 << uint(TCP_LISTEN))),
		States: TCP_ALL_FLAGS,

		ID: InetDiagSockID{
			// As seen on [0], there are no mentions to r->id.idiag_src or r->id.idiag_dst so it looks like
			// filtering on IP addresses has no effect whatsoever. The same goes for interfaces and cookies
			// as far as we can tell... On the other hand, if looking for a single socket these seem to be
			// taken into account [1].
			// 0: https://elixir.bootlin.com/linux/v6.12.4/source/net/ipv4/inet_diag.c#L1019
			// 1: https://elixir.bootlin.com/linux/v6.12.4/source/net/ipv4/inet_diag.c#L519
			// DiagSrc:   DiagIPT(srcIP),
			// DiagDst:   [16]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
			DiagSPort: [2]byte{byte(srcPort >> 8), byte(srcPort & 0xFF)},
			DiagDPort: [2]byte{0, 0},
		},
	}

	// Simply zero-out the 0xFF written to signal the 4-in-6 representation. However, the kernel
	// has several references to 4-in-6 lying around, so this might not be needed at all!
	// if family == unix.AF_INET {
	// 	msg.ID.DiagSrc[len(msg.ID.DiagSrc)-5] = 0
	// 	msg.ID.DiagSrc[len(msg.ID.DiagSrc)-6] = 0
	// }

	// We'll basically ask for all available information really...
	// Note the 1-offset induced (we think) by the fact that
	// netlink.INET_DIAG_NONE == 0. At any rate, be sure to check
	// [0] to see what flags influence the gathered information.
	// 0: https://elixir.bootlin.com/linux/v6.12.4/source/net/ipv4/inet_diag.c#L237
	msg.Ext |= (1 << (netlink.INET_DIAG_MEMINFO - 1))
	msg.Ext |= (1 << (netlink.INET_DIAG_INFO - 1))
	msg.Ext |= (1 << (netlink.INET_DIAG_VEGASINFO - 1))
	msg.Ext |= (1 << (netlink.INET_DIAG_CONG - 1))

	msg.Ext |= (1 << (netlink.INET_DIAG_TCLASS - 1))
	msg.Ext |= (1 << (netlink.INET_DIAG_TOS - 1))
	msg.Ext |= (1 << (netlink.INET_DIAG_SKMEMINFO - 1))
	msg.Ext |= (1 << (netlink.INET_DIAG_SHUTDOWN - 1))

	slog.Debug("crafted request", "msg", msg)

	req.AddData(msg)

	// These were included in tcp-info, but they might be redundant...
	// req.NlMsghdr.Type = nl.SOCK_DIAG_BY_FAMILY
	// req.NlMsghdr.Flags |= syscall.NLM_F_DUMP | syscall.NLM_F_REQUEST

	return req
}

// This is mostly pulled from the implementation of netlink.SocketTCPInfo
func (p *NetlinkPlugin) ExecuteRequest(req *nl.NetlinkRequest) error {
	slog.Debug("executing the netlink request")
	var (
		results []*InetDiagTCPInfoResp
		err     error
	)
	nParsed := 0
	err = req.ExecuteIter(unix.NETLINK_INET_DIAG, nl.SOCK_DIAG_BY_FAMILY, func(msg []byte) bool {
		sockInfo := &Socket{}
		if err = sockInfo.deserialize(msg); err != nil {
			slog.Error("couldn't parse socket information", "err", err)
			return false
		}
		// results = append(results, sockInfo)

		var attrs []syscall.NetlinkRouteAttr
		if attrs, err = nl.ParseRouteAttr(msg[sizeofSocket:]); err != nil {
			return false
		}

		var result *InetDiagTCPInfoResp
		if result, err = attrsToInetDiagTCPInfoResp(attrs, sockInfo); err != nil {
			slog.Error("error parsing netlink attributes", "err", err)
			return false
		}
		results = append(results, result)

		slog.Debug("parsing Netlink message", "nParsed", nParsed, "nAttrs", len(attrs))
		nParsed++

		return true
	})
	if err != nil {
		return fmt.Errorf("error executing the request: %w", err)
	}

	for i, result := range results {
		slog.Debug("gathered result", "i", i, "result", result)
	}
	return nil
	// return result, nil
}

func attrsToInetDiagTCPInfoResp(attrs []syscall.NetlinkRouteAttr, sockInfo *Socket) (*InetDiagTCPInfoResp, error) {
	info := &InetDiagTCPInfoResp{
		InetDiagMsg: sockInfo,
	}
	for _, a := range attrs {
		slog.Debug("parsing netlink attribute", "type", inetDiagMap[a.Attr.Type], "len", a.Attr.Len)
		switch a.Attr.Type {
		case netlink.INET_DIAG_INFO:
			info.TCPInfo = &TCPInfo{}
			if err := info.TCPInfo.deserialize(a.Value); err != nil {
				return nil, err
			}
		case netlink.INET_DIAG_BBRINFO:
			info.BBRInfo = &TCPBBRInfo{}
			if err := info.BBRInfo.deserialize(a.Value); err != nil {
				return nil, err
			}
		case netlink.INET_DIAG_TOS:
			info.TOS = &TOS{}
			if err := info.TOS.deserialize(a.Value); err != nil {
				return nil, err
			}
		case netlink.INET_DIAG_MEMINFO:
			info.MemInfo = &MemInfo{}
			if err := info.MemInfo.deserialize(a.Value); err != nil {
				return nil, err
			}
		case netlink.INET_DIAG_SKMEMINFO:
			info.SkMemInfo = &SkMemInfo{}
			if err := info.SkMemInfo.deserialize(a.Value); err != nil {
				return nil, err
			}
		case netlink.INET_DIAG_CONG:
			info.Cong = &Cong{}
			if err := info.Cong.deserialize(a.Value); err != nil {
				return nil, err
			}
		}
	}

	return info, nil
}

func (p *NetlinkPlugin) fTupleToFlowID(f fTuple, conn *netlink.Socket, state glowdTypes.FlowState) glowdTypes.FlowID {
	return glowdTypes.FlowID{
		State:    state,
		StartTs:  time.Now(),
		Protocol: glowdTypes.TCP,
		Src: glowdTypes.IPPort{
			IP:   conn.ID.Source,
			Port: conn.ID.SourcePort,
		},
		Dst: glowdTypes.IPPort{
			IP:   conn.ID.Destination,
			Port: conn.ID.DestinationPort,
		},
		Activity:   p.ActivityID,
		Experiment: p.ExperimentID,
	}
}

func (p *NetlinkPlugin) Run(done <-chan struct{}, outChan chan<- glowdTypes.FlowID) {
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

					outChan <- p.fTupleToFlowID(tmp, conn, glowdTypes.START)

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
