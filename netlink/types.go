package netlink

import (
	"fmt"
	"unsafe"

	"github.com/vishvananda/netlink"
)

// SockDiagReq is the Netlink request struct, as in linux/inet_diag.h
// Note that netlink messages use host byte ordering. This struct's
// definition has been pulled from github.com/m-lab/tcp-info. Check
// sock_diag(7) for more information.
type SockDiagReq struct {
	Family   uint8
	Protocol uint8
	Ext      uint8
	Pad      uint8
	States   uint32
	ID       InetDiagSockID
}

// SizeOfSockDiagReq is the size of the struct.
// TODO should we just make this explicit in the code?
const SizeOfSockDiagReq = int(unsafe.Sizeof(SockDiagReq{})) // Should be 0x38

// Len is required to implement the nl.NetlinkRequestData interface
func (req SockDiagReq) Len() int {
	return SizeOfSockDiagReq
}

// Serialize is required to implement the nl.NetlinkRequestData interface
func (req SockDiagReq) Serialize() []byte {
	return (*(*[SizeOfSockDiagReq]byte)(unsafe.Pointer(&req)))[:]
}

// InetDiagSockID is the binary linux representation of a socket, as in linux/inet_diag.h.
// This struct definition has been adapted from github.com/m-lab/tcp-info. This is basically
// the Golang counterpart of 'struct inet_diag_sockid'. Check sock_diag(7) for more information.
type InetDiagSockID struct {
	DiagSPort  DiagPortT
	DiagDPort  DiagPortT
	DiagSrc    DiagIPT
	DiagDst    DiagIPT
	DiagIf     DiagNetIfT
	DiagCookie DiagCookieT
}

// DiagPortT encodes an InetDiagSockID port.
type DiagPortT [2]byte

// DiagIPT encodes an InetDiagSockID IPv{4,6} address.
type DiagIPT [16]byte

// DiagNetIfT encodes an InetDiagSockID Interface field.
type DiagNetIfT [4]byte

// Types for LinuxSockID fields.
type DiagCookieT [8]byte

// Socket simply embeds netlink.Socket to allow us to expose the
// private deserialize method.
type Socket struct {
	netlink.Socket
}

func (i *Socket) String() string {
	return fmt.Sprintf("%#v", *i)
}

// SocketMemInfo implements the struct associated with INET_DIAG_SKMEMINFO,
// described only in section 'Socket memory information' of sock_diag(7).
type SocketMemInfo struct {
	RmemAlloc  uint32 `json:""`
	Rcvbuf     uint32 `json:""`
	WmemAlloc  uint32 `json:""`
	Sndbuf     uint32 `json:""`
	FwdAlloc   uint32 `json:""`
	WmemQueued uint32 `json:""`
	Optmem     uint32 `json:""`
	Backlog    uint32 `json:""`
	// Drops      uint32
}

// VegasInfo implements the struct associated with INET_DIAG_VEGASINFO, corresponding with
// linux struct tcpvegas_info in uapi/linux/inet_diag.h.
type VegasInfo struct {
	Enabled  uint32
	RTTCount uint32
	RTT      uint32
	MinRTT   uint32
}

// DCTCPInfo implements the struct associated with INET_DIAG_DCTCPINFO attribute, corresponding with
// linux struct tcp_dctcp_info in uapi/linux/inet_diag.h.
type DCTCPInfo struct {
	Enabled uint16
	CEState uint16
	Alpha   uint32
	ABEcn   uint32
	ABTot   uint32
}

// BBRInfo implements the struct associated with INET_DIAG_BBRINFO attribute, corresponding with
// linux struct tcp_bbr_info in uapi/linux/inet_diag.h.
type BBRInfo struct {
	BW         int64  // Max-filtered BW (app throughput) estimate in bytes/second
	MinRTT     uint32 // Min-filtered RTT in uSec
	PacingGain uint32 // Pacing gain shifted left 8 bits
	CwndGain   uint32 // Cwnd gain shifted left 8 bits
}

// State is the enumeration of TCP states.
// https://datatracker.ietf.org/doc/draft-ietf-tcpm-rfc793bis/
// and uapi/linux/tcp.h
type State int32

// All of these constants' names make the linter complain, but we inherited
// these names from external C code, so we will keep them.
const (
	TCP_INVALID     State = 0
	TCP_ESTABLISHED State = 1 // or unix.BPF_TCP_ESTABLISHED
	TCP_SYN_SENT    State = 2
	TCP_SYN_RECV    State = 3
	TCP_FIN_WAIT1   State = 4
	TCP_FIN_WAIT2   State = 5
	TCP_TIME_WAIT   State = 6
	TCP_CLOSE       State = 7
	TCP_CLOSE_WAIT  State = 8
	TCP_LAST_ACK    State = 9
	TCP_LISTEN      State = 10 // or unix.BPF_TCP_LISTEN
	TCP_CLOSING     State = 11

	// TCP_ALL_FLAGS includes flag bits for all TCP connection states. It corresponds to TCPF_ALL in some linux code.
	TCP_ALL_FLAGS = 0xFFF
)

var stateName = map[State]string{
	0:  "INVALID",
	1:  "ESTABLISHED",
	2:  "SYN_SENT",
	3:  "SYN_RECV",
	4:  "FIN_WAIT1",
	5:  "FIN_WAIT2",
	6:  "TIME_WAIT",
	7:  "CLOSE",
	8:  "CLOSE_WAIT",
	9:  "LAST_ACK",
	10: "LISTEN",
	11: "CLOSING",
}

func (x State) String() string {
	s, ok := stateName[x]
	if !ok {
		return fmt.Sprintf("UNKNOWN_STATE_%d", x)
	}
	return s
}

// LinuxTCPInfo is the linux defined structure returned in RouteAttr DIAG_INFO messages.
// It corresponds to the struct tcp_info in [0]. This struct definition has been plundered
// from github.com/m-lab/tcp-info.
// References:
//
//	0: https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/include/uapi/linux/tcp.h
type LinuxTCPInfo struct {
	State       uint8
	CAState     uint8
	Retransmits uint8
	Probes      uint8
	Backoff     uint8
	Options     uint8
	WScale      uint8 //snd_wscale : 4, tcpi_rcv_wscale : 4;
	AppLimited  uint8 //delivery_rate_app_limited:1;

	RTO    uint32 // offset 8
	ATO    uint32
	SndMSS uint32
	RcvMSS uint32

	Unacked uint32 // offset 24
	Sacked  uint32
	Lost    uint32
	Retrans uint32
	Fackets uint32

	/* Times. */
	// These seem to be elapsed time, so they increase on almost every sample.
	// We can probably use them to get more info about intervals between samples.
	LastDataSent uint32 // offset 44
	LastAckSent  uint32 /* Not remembered, sorry. */ // offset 48
	LastDataRecv uint32 // offset 52
	LastAckRecv  uint32 // offset 56

	/* Metrics. */
	PMTU        uint32
	RcvSsThresh uint32
	RTT         uint32
	RTTVar      uint32
	SndSsThresh uint32
	SndCwnd     uint32
	AdvMSS      uint32
	Reordering  uint32

	RcvRTT   uint32
	RcvSpace uint32

	TotalRetrans uint32

	PacingRate    int64 // This is often -1, so better for it to be signed
	MaxPacingRate int64 // This is often -1, so better to be signed.

	// NOTE: In linux, these are uint64, but we make them int64 here for compatibility with BigQuery
	BytesAcked    int64 /* RFC4898 tcpEStatsAppHCThruOctetsAcked */
	BytesReceived int64 /* RFC4898 tcpEStatsAppHCThruOctetsReceived */
	SegsOut       int32 /* RFC4898 tcpEStatsPerfSegsOut */
	SegsIn        int32 /* RFC4898 tcpEStatsPerfSegsIn */

	NotsentBytes uint32
	MinRTT       uint32
	DataSegsIn   uint32 /* RFC4898 tcpEStatsDataSegsIn */
	DataSegsOut  uint32 /* RFC4898 tcpEStatsDataSegsOut */

	// NOTE: In linux, this is uint64, but we make it int64 here for compatibility with BigQuery
	DeliveryRate int64

	BusyTime      int64 /* Time (usec) busy sending data */
	RWndLimited   int64 /* Time (usec) limited by receive window */
	SndBufLimited int64 /* Time (usec) limited by send buffer */

	Delivered   uint32
	DeliveredCE uint32

	// NOTE: In linux, these are uint64, but we make them int64 here for compatibility with BigQuery
	BytesSent    int64 /* RFC4898 tcpEStatsPerfHCDataOctetsOut */
	BytesRetrans int64 /* RFC4898 tcpEStatsPerfOctetsRetrans */

	DSackDups uint32 /* RFC4898 tcpEStatsStackDSACKDups */
	ReordSeen uint32 /* reordering events seen */

	RcvOooPack uint32 /* Out-of-order packets received */

	SndWnd uint32 /* peer's advertised receive window after scaling (bytes) */
}

// This is pulled from netlink and should be equivalent to the above!
type TCPInfo struct {
	State                     uint8  `json:"state"`
	Ca_state                  uint8  `json:"caState"`
	Retransmits               uint8  `json:"retransmits"`
	Probes                    uint8  `json:"probes"`
	Backoff                   uint8  `json:"backoff"`
	Options                   uint8  `json:"options"`
	Snd_wscale                uint8  `json:"sndWscale"` // no uint4
	Rcv_wscale                uint8  `json:"rcvdWscale"`
	Delivery_rate_app_limited uint8  `json:"deliveryRateAppLimited"`
	Fastopen_client_fail      uint8  `json:"fastOpenClientFail"`
	Rto                       uint32 `json:"rto"`
	Ato                       uint32 `json:"ato"`
	Snd_mss                   uint32 `json:"sndMss"`
	Rcv_mss                   uint32 `json:"rcvMss"`
	Unacked                   uint32 `json:"unAcked"`
	Sacked                    uint32 `json:"sAcked"`
	Lost                      uint32 `json:"lost"`
	Retrans                   uint32 `json:"retrans"`
	Fackets                   uint32 `json:"fAckets"`
	Last_data_sent            uint32 `json:"lastDataSent"`
	Last_ack_sent             uint32 `json:"lastAckSent"`
	Last_data_recv            uint32 `json:"lastDataRecv"`
	Last_ack_recv             uint32 `json:"lastAckRecv"`
	Pmtu                      uint32 `json:"pMtu"`
	Rcv_ssthresh              uint32 `json:"rcvSsThresh"`
	Rtt                       uint32 `json:"rtt"`
	Rttvar                    uint32 `json:"rttVar"`
	Snd_ssthresh              uint32 `json:"sndSsThresh"`
	Snd_cwnd                  uint32 `json:"sndCwnd"`
	Advmss                    uint32 `json:"advMss"`
	Reordering                uint32 `json:"reordering"`
	Rcv_rtt                   uint32 `json:"rcvRtt"`
	Rcv_space                 uint32 `json:"rcvSpace"`
	Total_retrans             uint32 `json:"totalRetrans"`
	Pacing_rate               uint64 `json:"pacingRate"`
	Max_pacing_rate           uint64 `json:"maxPacingRate"`
	Bytes_acked               uint64 `json:"bytesAcked"` /* RFC4898 tcpEStatsAppHCThruOctetsAcked */
	Bytes_received            uint64 `json:"bytesRecv"`  /* RFC4898 tcpEStatsAppHCThruOctetsReceived */
	Segs_out                  uint32 `json:"segsOut"`    /* RFC4898 tcpEStatsPerfSegsOut */
	Segs_in                   uint32 `json:"segsIn"`     /* RFC4898 tcpEStatsPerfSegsIn */
	Notsent_bytes             uint32 `json:"notsentBytes"`
	Min_rtt                   uint32 `json:"minRtt"`
	Data_segs_in              uint32 `json:"dataSegsIn"`  /* RFC4898 tcpEStatsDataSegsIn */
	Data_segs_out             uint32 `json:"dataSegsOut"` /* RFC4898 tcpEStatsDataSegsOut */
	Delivery_rate             uint64 `json:"deliveryRate"`
	Busy_time                 uint64 `json:"busyTime"`      /* Time (usec) busy sending data */
	Rwnd_limited              uint64 `json:"rwndLimited"`   /* Time (usec) limited by receive window */
	Sndbuf_limited            uint64 `json:"sndBufLimited"` /* Time (usec) limited by send buffer */
	Delivered                 uint32 `json:"delivered"`
	Delivered_ce              uint32 `json:"deliveredCe"`
	Bytes_sent                uint64 `json:"bytesSent"`    /* RFC4898 tcpEStatsPerfHCDataOctetsOut */
	Bytes_retrans             uint64 `json:"bytesRetrans"` /* RFC4898 tcpEStatsPerfOctetsRetrans */
	Dsack_dups                uint32 `json:"dsAckDups"`    /* RFC4898 tcpEStatsStackDSACKDups */
	Reord_seen                uint32 `json:"reordSeen"`    /* reordering events seen */
	Rcv_ooopack               uint32 `json:"rcvOoopack"`   /* Out-of-order packets received */
	Snd_wnd                   uint32 `json:"sndWnd"`       /* peer's advertised receive window after * scaling (bytes) */
}

func (i *TCPInfo) String() string {
	return fmt.Sprintf("%#v", *i)
}

type InetDiagTCPInfoResp struct {
	InetDiagMsg *Socket     `json:"skBuff"`
	TCPInfo     *TCPInfo    `json:"tcpInfo"`
	BBRInfo     *TCPBBRInfo `json:"bbr"`
	TOS         *TOS        `json:"tos"`
	MemInfo     *MemInfo    `json:"memInfo"`
	SkMemInfo   *SkMemInfo  `json:"skMemInfo"`
	Cong        *Cong       `json:"cong"`
}

type Cong struct {
	Algorithm string `json:"algorithm"`
}

func (i *Cong) String() string {
	return i.Algorithm
}

type TOS struct {
	TOS uint8 `json:"tos"`
}

func (i *TOS) String() string {
	return fmt.Sprintf("%#v", *i)
}

// Taken from sock_diag(7)
type SkMemInfo struct {
	// The amount of data in receive queue.
	RMemAlloc uint32 `json:"rMemAlloc"`
	// The receive socket buffer as set by SO_RCVBUF.
	RcvBuff uint32 `json:"rcvBuff"`
	// The amount of data in send queue.
	WMemAlloc uint32 `json:"WMemAlloc"`
	// The send socket buffer as set by SO_SNDBUF.
	SndBuff uint32 `json:"sndBuff"`
	// The amount of memory scheduled for future use (TCP only).
	FwdAlloc uint32 `json:"fwdAlloc"`
	// The amount of data queued by TCP, but not yet sent.
	WMemQueued uint32 `json:"wMemQueued"`
	// The amount of memory allocated for the socket's service needs (e.g., socket filter).
	OptMem uint32 `json:"optMem"`
	// The amount of packets in the backlog (not yet processed).
	Backlog uint32 `json:"backlog"`
	// Check https://manpages.debian.org/stretch/manpages/sock_diag.7.en.html
	Drops uint32 `json:"drops"`
}

func (i *SkMemInfo) String() string {
	return fmt.Sprintf("%#v", *i)
}

type TCPBBRInfo struct {
	BBRBW         uint64 `json:"bbrBW"`
	BBRMinRTT     uint32 `json:"bbrMinRTT"`
	BBRPacingGain uint32 `json:"bbrPacingGain"`
	BBRCwndGain   uint32 `json:"bbrCwndGain"`
}

// MemInfo implements the struct associated with INET_DIAG_MEMINFO, corresponding with
// linux struct inet_diag_meminfo in uapi/linux/inet_diag.h.
type MemInfo struct {
	RMem uint32 `json:"rMem"`
	WMem uint32 `json:"wMem"`
	FMem uint32 `json:"fMem"`
	TMem uint32 `json:"tMem"`
}

func (i *MemInfo) String() string {
	return fmt.Sprintf("%#v", *i)
}

var inetDiagMap = map[uint16]string{
	netlink.INET_DIAG_NONE:            "INET_DIAG_NONE",
	netlink.INET_DIAG_MEMINFO:         "INET_DIAG_MEMINFO",
	netlink.INET_DIAG_INFO:            "INET_DIAG_INFO",
	netlink.INET_DIAG_VEGASINFO:       "INET_DIAG_VEGASINFO",
	netlink.INET_DIAG_CONG:            "INET_DIAG_CONG",
	netlink.INET_DIAG_TOS:             "INET_DIAG_TOS",
	netlink.INET_DIAG_TCLASS:          "INET_DIAG_TCLASS",
	netlink.INET_DIAG_SKMEMINFO:       "INET_DIAG_SKMEMINFO",
	netlink.INET_DIAG_SHUTDOWN:        "INET_DIAG_SHUTDOWN",
	netlink.INET_DIAG_DCTCPINFO:       "INET_DIAG_DCTCPINFO",
	netlink.INET_DIAG_PROTOCOL:        "INET_DIAG_PROTOCOL",
	netlink.INET_DIAG_SKV6ONLY:        "INET_DIAG_SKV6ONLY",
	netlink.INET_DIAG_LOCALS:          "INET_DIAG_LOCALS",
	netlink.INET_DIAG_PEERS:           "INET_DIAG_PEERS",
	netlink.INET_DIAG_PAD:             "INET_DIAG_PAD",
	netlink.INET_DIAG_MARK:            "INET_DIAG_MARK",
	netlink.INET_DIAG_BBRINFO:         "INET_DIAG_BBRINFO",
	netlink.INET_DIAG_CLASS_ID:        "INET_DIAG_CLASS_ID",
	netlink.INET_DIAG_MD5SIG:          "INET_DIAG_MD5SIG",
	netlink.INET_DIAG_ULP_INFO:        "INET_DIAG_ULP_INFO",
	netlink.INET_DIAG_SK_BPF_STORAGES: "INET_DIAG_SK_BPF_STORAGES",
	netlink.INET_DIAG_CGROUP_ID:       "INET_DIAG_CGROUP_ID",
	netlink.INET_DIAG_SOCKOPT:         "INET_DIAG_SOCKOPT",
	netlink.INET_DIAG_MAX:             "INET_DIAG_MAX",
}
