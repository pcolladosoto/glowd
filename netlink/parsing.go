package netlink

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"unicode"

	"github.com/vishvananda/netlink/nl"
	"golang.org/x/sys/unix"
)

const (
	sizeofSocketID      = 0x30
	sizeofSocketRequest = sizeofSocketID + 0x8
	sizeofSocket        = sizeofSocketID + 0x18
)

var (
	native       = nl.NativeEndian()
	networkOrder = binary.BigEndian
)

// Please not the readBuffer has been plundered from
// github.com/vishvananda/netlink/socket_linux.go
type readBuffer struct {
	Bytes []byte
	pos   int
}

func (b *readBuffer) Read() byte {
	c := b.Bytes[b.pos]
	b.pos++
	return c
}

func (b *readBuffer) Next(n int) []byte {
	s := b.Bytes[b.pos : b.pos+n]
	b.pos += n
	return s
}

func (s *Socket) deserialize(b []byte) error {
	if len(b) < sizeofSocket {
		return fmt.Errorf("socket data short read (%d); want %d", len(b), sizeofSocket)
	}

	rb := readBuffer{Bytes: b}
	s.Family = rb.Read()
	s.State = rb.Read()
	s.Timer = rb.Read()
	s.Retrans = rb.Read()
	s.ID.SourcePort = networkOrder.Uint16(rb.Next(2))
	s.ID.DestinationPort = networkOrder.Uint16(rb.Next(2))
	if s.Family == unix.AF_INET6 {
		s.ID.Source = net.IP(rb.Next(16))
		s.ID.Destination = net.IP(rb.Next(16))
	} else {
		s.ID.Source = net.IPv4(rb.Read(), rb.Read(), rb.Read(), rb.Read())
		rb.Next(12)
		s.ID.Destination = net.IPv4(rb.Read(), rb.Read(), rb.Read(), rb.Read())
		rb.Next(12)
	}
	s.ID.Interface = native.Uint32(rb.Next(4))
	s.ID.Cookie[0] = native.Uint32(rb.Next(4))
	s.ID.Cookie[1] = native.Uint32(rb.Next(4))
	s.Expires = native.Uint32(rb.Next(4))
	s.RQueue = native.Uint32(rb.Next(4))
	s.WQueue = native.Uint32(rb.Next(4))
	s.UID = native.Uint32(rb.Next(4))
	s.INode = native.Uint32(rb.Next(4))

	return nil
}

const (
	tcpBBRInfoLen = 20
	memInfoLen    = 16
	tosLen        = 1
	skMemInfoLen  = 36
)

func (t *TOS) deserialize(b []byte) error {
	if len(b) != tosLen {
		return errors.New("Invalid length")
	}

	var err error
	rb := bytes.NewBuffer(b)
	t.TOS, err = rb.ReadByte()
	if err != nil {
		return checkDeserErr(err)
	}
	return nil
}

func checkDeserErr(err error) error {
	if err == io.EOF {
		return nil
	}
	return err
}

func (t *Cong) deserialize(b []byte) error {
	// Drop the trailing '\0
	s := string(b[:len(b)-1])
	for _, c := range s {
		if c > unicode.MaxASCII {
			return errors.New("non-ASCII character found")
		}
	}

	t.Algorithm = s
	return nil
}

func (t *VegasInfo) deserialize(b []byte) error {
	if len(b) != 16 {
		return errors.New("invalid length for vegasInfo")
	}

	rb := bytes.NewBuffer(b)

	next := rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Enabled = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.RTTCount = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.RTT = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.MinRTT = native.Uint32(next)

	return nil
}

func (t *DCTCPInfo) deserialize(b []byte) error {
	if len(b) != 16 {
		return errors.New("invalid length for dctcpInfo")
	}

	rb := bytes.NewBuffer(b)

	next := rb.Next(2)
	if len(next) == 0 {
		return nil
	}
	t.Enabled = native.Uint16(next)

	next = rb.Next(2)
	if len(next) == 0 {
		return nil
	}
	t.CEState = native.Uint16(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Alpha = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.ABEcn = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.ABTot = native.Uint32(next)

	return nil
}

func (t *SkMemInfo) deserialize(b []byte) error {
	if len(b) != skMemInfoLen {
		return errors.New("Invalid length")
	}

	rb := bytes.NewBuffer(b)

	next := rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.RMemAlloc = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.RcvBuff = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.WMemAlloc = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.SndBuff = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.FwdAlloc = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.WMemQueued = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.OptMem = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Backlog = native.Uint32(next)

	return nil
}

func (t *TCPInfo) deserialize(b []byte) error {
	var err error
	rb := bytes.NewBuffer(b)

	t.State, err = rb.ReadByte()
	if err != nil {
		return checkDeserErr(err)
	}

	t.Ca_state, err = rb.ReadByte()
	if err != nil {
		return checkDeserErr(err)
	}

	t.Retransmits, err = rb.ReadByte()
	if err != nil {
		return checkDeserErr(err)
	}

	t.Probes, err = rb.ReadByte()
	if err != nil {
		return checkDeserErr(err)
	}

	t.Backoff, err = rb.ReadByte()
	if err != nil {
		return checkDeserErr(err)
	}
	t.Options, err = rb.ReadByte()
	if err != nil {
		return checkDeserErr(err)
	}

	scales, err := rb.ReadByte()
	if err != nil {
		return checkDeserErr(err)
	}
	t.Snd_wscale = scales >> 4  // first 4 bits
	t.Rcv_wscale = scales & 0xf // last 4 bits

	rateLimAndFastOpen, err := rb.ReadByte()
	if err != nil {
		return checkDeserErr(err)
	}
	t.Delivery_rate_app_limited = rateLimAndFastOpen >> 7 // get first bit
	t.Fastopen_client_fail = rateLimAndFastOpen >> 5 & 3  // get next two bits

	next := rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Rto = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Ato = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Snd_mss = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Rcv_mss = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Unacked = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Sacked = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Lost = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Retrans = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Fackets = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Last_data_sent = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Last_ack_sent = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Last_data_recv = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Last_ack_recv = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Pmtu = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Rcv_ssthresh = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Rtt = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Rttvar = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Snd_ssthresh = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Snd_cwnd = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Advmss = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Reordering = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Rcv_rtt = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Rcv_space = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Total_retrans = native.Uint32(next)

	next = rb.Next(8)
	if len(next) == 0 {
		return nil
	}
	t.Pacing_rate = native.Uint64(next)

	next = rb.Next(8)
	if len(next) == 0 {
		return nil
	}
	t.Max_pacing_rate = native.Uint64(next)

	next = rb.Next(8)
	if len(next) == 0 {
		return nil
	}
	t.Bytes_acked = native.Uint64(next)

	next = rb.Next(8)
	if len(next) == 0 {
		return nil
	}
	t.Bytes_received = native.Uint64(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Segs_out = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Segs_in = native.Uint32(next)
	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Notsent_bytes = native.Uint32(next)
	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Min_rtt = native.Uint32(next)
	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Data_segs_in = native.Uint32(next)
	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Data_segs_out = native.Uint32(next)

	next = rb.Next(8)
	if len(next) == 0 {
		return nil
	}
	t.Delivery_rate = native.Uint64(next)

	next = rb.Next(8)
	if len(next) == 0 {
		return nil
	}
	t.Busy_time = native.Uint64(next)

	next = rb.Next(8)
	if len(next) == 0 {
		return nil
	}
	t.Rwnd_limited = native.Uint64(next)

	next = rb.Next(8)
	if len(next) == 0 {
		return nil
	}
	t.Sndbuf_limited = native.Uint64(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Delivered = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Delivered_ce = native.Uint32(next)

	next = rb.Next(8)
	if len(next) == 0 {
		return nil
	}
	t.Bytes_sent = native.Uint64(next)

	next = rb.Next(8)
	if len(next) == 0 {
		return nil
	}
	t.Bytes_retrans = native.Uint64(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Dsack_dups = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Reord_seen = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Rcv_ooopack = native.Uint32(next)

	next = rb.Next(4)
	if len(next) == 0 {
		return nil
	}
	t.Snd_wnd = native.Uint32(next)
	return nil
}

func (t *TCPBBRInfo) deserialize(b []byte) error {
	if len(b) != tcpBBRInfoLen {
		return errors.New("Invalid length")
	}

	rb := bytes.NewBuffer(b)
	t.BBRBW = native.Uint64(rb.Next(8))
	t.BBRMinRTT = native.Uint32(rb.Next(4))
	t.BBRPacingGain = native.Uint32(rb.Next(4))
	t.BBRCwndGain = native.Uint32(rb.Next(4))

	return nil
}

func (m *MemInfo) deserialize(b []byte) error {
	if len(b) != memInfoLen {
		return errors.New("Invalid length")
	}

	rb := bytes.NewBuffer(b)
	m.RMem = native.Uint32(rb.Next(4))
	m.WMem = native.Uint32(rb.Next(4))
	m.FMem = native.Uint32(rb.Next(4))
	m.TMem = native.Uint32(rb.Next(4))

	return nil
}
