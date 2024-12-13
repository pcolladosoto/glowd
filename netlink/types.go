package netlink

import (
	"fmt"
	"unsafe"

	"github.com/vishvananda/netlink"
)

// State is the enumeration of TCP states.
// https://datatracker.ietf.org/doc/draft-ietf-tcpm-rfc793bis/
// and uapi/linux/tcp.h
type State int32

func (x State) String() string {
	s, ok := stateName[x]
	if !ok {
		return fmt.Sprintf("UNKNOWN_STATE_%d", x)
	}
	return s
}

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

type InetDiagTCPInfoResp struct {
	InetDiagMsg *Socket     `json:"skBuff"`
	TCPInfo     *TCPInfo    `json:"tcpInfo"`
	BBRInfo     *TCPBBRInfo `json:"bbr"`
	TOS         *TOS        `json:"tos"`
	MemInfo     *MemInfo    `json:"memInfo"`
	SkMemInfo   *SkMemInfo  `json:"skMemInfo"`
	Cong        *Cong       `json:"cong"`
	VegasInfo   *VegasInfo  `json:"vegasInfo,omitempty"`
	DCTCPInfo   *DCTCPInfo  `json:"dctcpInfo,omitempty"`
}

// Socket simply embeds netlink.Socket to allow us to expose the
// private deserialize method.
type Socket struct {
	netlink.Socket
}

func (i *Socket) String() string {
	return fmt.Sprintf("%#v", *i)
}

// VegasInfo implements the struct associated with INET_DIAG_VEGASINFO, corresponding with
// linux struct tcpvegas_info in uapi/linux/inet_diag.h.
type VegasInfo struct {
	Enabled  uint32 `json:"enabled"`
	RTTCount uint32 `json:"rttCount"`
	RTT      uint32 `json:"rtt"`
	MinRTT   uint32 `json:"minRtt"`
}

func (i *VegasInfo) String() string {
	return fmt.Sprintf("%#v", *i)
}

// DCTCPInfo implements the struct associated with INET_DIAG_DCTCPINFO attribute, corresponding with
// linux struct tcp_dctcp_info in uapi/linux/inet_diag.h.
type DCTCPInfo struct {
	Enabled uint16 `json:"enabled"`
	CEState uint16 `json:"ceState"`
	Alpha   uint32 `json:"alpha"`
	ABEcn   uint32 `json:"abeCn"`
	ABTot   uint32 `json:"abTot"`
}

func (i *DCTCPInfo) String() string {
	return fmt.Sprintf("%#v", *i)
}

// TCPInfo is the linux defined structure returned in RouteAttr INET_DIAG_INFO messages.
// It corresponds to the struct tcp_info in [0]. This struct definition has been plundered
// from github.com/m-lab/tcp-info.
// References:
//
//	0: https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/include/uapi/linux/tcp.h
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

	/* Times. */
	// These seem to be elapsed time, so they increase on almost every sample.
	// We can probably use them to get more info about intervals between samples.
	Last_data_sent uint32 `json:"lastDataSent"`
	Last_ack_sent  uint32 `json:"lastAckSent"`
	Last_data_recv uint32 `json:"lastDataRecv"`
	Last_ack_recv  uint32 `json:"lastAckRecv"`

	/* Metrics. */
	Pmtu            uint32 `json:"pMtu"`
	Rcv_ssthresh    uint32 `json:"rcvSsThresh"`
	Rtt             uint32 `json:"rtt"`
	Rttvar          uint32 `json:"rttVar"`
	Snd_ssthresh    uint32 `json:"sndSsThresh"`
	Snd_cwnd        uint32 `json:"sndCwnd"`
	Advmss          uint32 `json:"advMss"`
	Reordering      uint32 `json:"reordering"`
	Rcv_rtt         uint32 `json:"rcvRtt"`
	Rcv_space       uint32 `json:"rcvSpace"`
	Total_retrans   uint32 `json:"totalRetrans"`
	Pacing_rate     uint64 `json:"pacingRate"`
	Max_pacing_rate uint64 `json:"maxPacingRate"`

	/* RFC4898 tcpEStatsAppHCThruOctetsAcked */
	Bytes_acked uint64 `json:"bytesAcked"`
	/* RFC4898 tcpEStatsAppHCThruOctetsReceived */
	Bytes_received uint64 `json:"bytesRecv"`
	/* RFC4898 tcpEStatsPerfSegsOut */
	Segs_out uint32 `json:"segsOut"`
	/* RFC4898 tcpEStatsPerfSegsIn */
	Segs_in       uint32 `json:"segsIn"`
	Notsent_bytes uint32 `json:"notsentBytes"`
	Min_rtt       uint32 `json:"minRtt"`
	/* RFC4898 tcpEStatsDataSegsIn */
	Data_segs_in uint32 `json:"dataSegsIn"`
	/* RFC4898 tcpEStatsDataSegsOut */
	Data_segs_out uint32 `json:"dataSegsOut"`

	Delivery_rate uint64 `json:"deliveryRate"`

	/* Time (usec) busy sending data */
	Busy_time uint64 `json:"busyTime"`
	/* Time (usec) limited by receive window */
	Rwnd_limited uint64 `json:"rwndLimited"`
	/* Time (usec) limited by send buffer */
	Sndbuf_limited uint64 `json:"sndBufLimited"`

	Delivered    uint32 `json:"delivered"`
	Delivered_ce uint32 `json:"deliveredCe"`

	/* RFC4898 tcpEStatsPerfHCDataOctetsOut */
	Bytes_sent uint64 `json:"bytesSent"`
	/* RFC4898 tcpEStatsPerfOctetsRetrans */
	Bytes_retrans uint64 `json:"bytesRetrans"`
	/* RFC4898 tcpEStatsStackDSACKDups */
	Dsack_dups uint32 `json:"dsAckDups"`
	/* reordering events seen */
	Reord_seen uint32 `json:"reordSeen"`
	/* Out-of-order packets received */
	Rcv_ooopack uint32 `json:"rcvOooPack"`
	/* peer's advertised receive window after scaling (bytes) */
	Snd_wnd uint32 `json:"sndWnd"`
}

func (i *TCPInfo) String() string {
	return fmt.Sprintf("%#v", *i)
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

// TCPBBRInfo implements the struct associated with INET_DIAG_BBRINFO attribute, corresponding with
// linux struct tcp_bbr_info in uapi/linux/inet_diag.h.
type TCPBBRInfo struct {
	// Max-filtered BW (app throughput) estimate in bytes/second
	BBRBW uint64 `json:"bbrBW"`
	// Min-filtered RTT in uSec
	BBRMinRTT uint32 `json:"bbrMinRTT"`
	// Pacing gain shifted left 8 bits
	BBRPacingGain uint32 `json:"bbrPacingGain"`
	// Cwnd gain shifted left 8 bits
	BBRCwndGain uint32 `json:"bbrCwndGain"`
}

func (i *TCPBBRInfo) String() string {
	return fmt.Sprintf("%#v", *i)
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
