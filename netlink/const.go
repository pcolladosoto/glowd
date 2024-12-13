package netlink

import "github.com/vishvananda/netlink"

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

var (
	stateName = map[State]string{
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

	inetDiagMap = map[uint16]string{
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
)
