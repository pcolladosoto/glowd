// +build ignore

/*
 * The authoritative reference is the Kernel documentation over at [0].
 * Be sure to check bpf-helpers(7) [1] too! This implementation is also
 * largely based on the example from libbpf-bootstrap [3]. Some great
 * documentation is also available over at [4]. Bear in mind this program
 * is of type BPF_PROG_TYPE_SCHED_CLS and we'll be running in the so called
 * direct action mode. What this implies togetehr with much more information
 * can be found on [5].
 * References:
 *   0: https://www.kernel.org/doc/html/latest/bpf/index.html
 *   1: https://www.man7.org/linux/man-pages/man7/bpf-helpers.7.html
 *   3: https://github.com/libbpf/libbpf-bootstrap/blob/4a567f229efe8fc79ee1a2249569eb6b9c02ad1b/examples/c/tc.bpf.c
 *   4: https://docs.ebpf.io/linux/helper-function/
 *   5: https://docs.ebpf.io/linux/program-type/BPF_PROG_TYPE_SCHED_CLS/
 */

// You'll need to install libbpf-devel (or the equivalent one) to get these headers!
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>
#include <bpf/bpf_tracing.h>

// Enforceable actions. These are defined on include/uapi/linux/if_ether.h
// (i.e. /usr/include/linux/pkt_cls.h). The problem is including linux/pkt_cls.h
// conflicts with the inclusion of vmlinux.h!
#define TC_ACT_UNSPEC (-1)
#define TC_ACT_OK 0

// The ETH_P_* constants are defined in include/uapi/linux/if_ether.h
// (i.e. /usr/include/linux/if_ether.h). Again, their inclusion conflicts
// with vmlinux.h...
#define ETH_P_IP    0x0800 /* Internet Protocol packet */
#define ETH_P_IPV6  0x86DD /* IPv6 over bluebook */
#define ETH_P_8021Q 0x8100 /* 802.1Q VLAN Extended Header */

// Protocol numbers. Check https://www.iana.org/assignments/protocol-numbers/protocol-numbers.xhtml
#define PROTO_IP_ICMP   0x01
#define PROTO_TCP       0x06
#define PROTO_UDP       0x11
#define PROTO_IPV6_ICMP 0x3A

// Oh wow, the kernel refuses to load unlicensed stuff!
char LICENSE[] SEC("license") = "GPL";

// The keys for our hash maps. Should we maybe combine the ports into a __u32?
struct fourTuple {
	__u64 ip6_hi;
	__u64 ip6_lo;
	__u16 dport;
	__u16 sport;
};

// Let's define our map!
struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__uint(max_entries, 100000);
	__type(key, struct fourTuple);
	__type(value, __u64);
} flowLabels SEC(".maps");

// long ringBufferFlags = 0;

// Let's hook the program on the TC! XDP will only look at the ingress traffic :(
SEC("classifier")

/*
 * The program will receive an __sk_buff which is a 'mirror' of the kernel's own
 * sk_buff [0]. This struct is very well documented over at [1].
 * References:
 *   0: https://docs.kernel.org/networking/skbuff.html
 *   1: https://docs.ebpf.io/linux/program-context/__sk_buff
*/
int target(struct __sk_buff *ctx) {
	void *data_end = (void *)(__u64)ctx->data_end;
	void *data = (void *)(__u64)ctx->data;

	// Check vmlinux.h for the definitions of these structs!
	// Also, the struct for 802.1Q seems to be vlan_ethhdr!
	struct ethhdr *l2;
	struct ipv6hdr *l3;
	struct tcphdr *l4;

	// Let's check whether the contents of the Ethernet frame are an IPv6 datagram.
	// We'll also need to be careful with the network's endianness, hence the call
	// to bpf_htons. This helper function is defined on libbpf's bpf_endian.h.
	if (ctx->protocol != bpf_htons(ETH_P_IPV6))
		return TC_ACT_OK;

	// Check https://docs.ebpf.io/linux/helper-function/bpf_trace_printk/
	// bpf_printk("hello from glowd's eBPF backend: we got an IPv6 datagram!");

	// Get a hold of the Ethernet frame header. We'll check we do indeed have more
	// information to read before going on.
	l2 = data;
	if ((void *)(l2 + 1) > data_end)
		return TC_ACT_OK;

	// Get a hold of the IPv6 header and check we do have some payload!
	l3 = (void *)(l2 + 1);
	if ((void *)(l3 + 1) > data_end)
		return TC_ACT_OK;

	__u64 ipv6SaddrLo = bpf_htonl(l3->saddr.in6_u.u6_addr32[2]);
	ipv6SaddrLo = ipv6SaddrLo << 32 | bpf_htonl(l3->saddr.in6_u.u6_addr32[3]);

	__u64 ipv6SaddrHi = bpf_htonl(l3->saddr.in6_u.u6_addr32[0]);
	ipv6SaddrHi = ipv6SaddrHi << 32 | bpf_htonl(l3->saddr.in6_u.u6_addr32[1]);

	__u64 ipv6DaddrLo = bpf_htonl(l3->daddr.in6_u.u6_addr32[2]);
	ipv6DaddrLo = ipv6DaddrLo << 32 | bpf_htonl(l3->daddr.in6_u.u6_addr32[3]);

	__u64 ipv6DaddrHi = bpf_htonl(l3->daddr.in6_u.u6_addr32[0]);
	ipv6DaddrHi = ipv6DaddrHi << 32 | bpf_htonl(l3->daddr.in6_u.u6_addr32[1]);

	__u8 flowLblLo = l3->flow_lbl[2];
	__u8 flowLblMi = l3->flow_lbl[1];
	__u8 flowLblHi = l3->flow_lbl[0];

	if (l3->nexthdr == PROTO_IPV6_ICMP) {
		bpf_printk("   IPv6 saddr: %pI6", &l3->saddr);
		bpf_printk("   IPv6 daddr: %pI6", &l3->daddr);

		bpf_printk("   IPv6 saddr (hi --- lo): %x --- %x", ipv6SaddrHi, ipv6SaddrLo);
		bpf_printk("   IPv6 daddr (hi --- lo): %x --- %x", ipv6DaddrHi, ipv6DaddrLo);

		bpf_printk("IPv6 flow_lbl: %x --- %x --- %x", flowLblHi, flowLblMi, flowLblLo);

		l3->flow_lbl[2] = 0xEF;
		l3->flow_lbl[1] = 0xCD;
		l3->flow_lbl[0] = 0xAB;

		return TC_ACT_OK;
	}

	if (l3->nexthdr == PROTO_TCP) {
		l4 = (void *)(l3 + 1);
		if ((void *)(l4 + 1) > data_end)
			return TC_ACT_OK;

		bpf_printk("TCP source: %x", bpf_htons(l4->source));
		bpf_printk("  TCP dest: %x", bpf_htons(l4->dest));
	}

	// At this point we have access to the full IPv6 header and payload. Time to mark the packet!

	// Simply signal that the packet should proceed!
	return TC_ACT_OK;
}

// BPF_HASH(flowlabel_table, struct fourtuple, u64, 100000);
// BPF_HASH(tobedeleted, struct fourtuple, u64, 100000);

// int set_flow_label(struct __sk_buff *skb) {
// 	u8 *cursor = 0;
// 	struct ethernet_t *ethernet = cursor_advance(cursor, sizeof(*ethernet));

// 	// IPv6
// 	if (ethernet->type == 0x86DD) {
// 		struct ip6_t *ip6 = cursor_advance(cursor, sizeof(*ip6));

// 		struct fourtuple addrport;

// 		// This is necessary for some reason to do with compiler padding
// 		__builtin_memset(&addrport, 0, sizeof(addrport));

// 		addrport.ip6_hi = ip6->dst_hi;
// 		addrport.ip6_lo = ip6->dst_lo;

// 		// TCP
// 		if (ip6->next_header == 6) {
// 			struct tcp_t *tcp = cursor_advance(cursor, sizeof(*tcp));

// 			addrport.dport = tcp->dst_port;
// 			addrport.sport = tcp->src_port;

// 			u64 *delete = tobedeleted.lookup(&addrport);

// 			u64 *flowlabel = flowlabel_table.lookup(&addrport);

// 			if (delete) {
// 				flowlabel_table.delete(&addrport);
// 				tobedeleted.delete(&addrport);
// 			}
// 			else if (flowlabel) {
// 				ip6->flow_label = *flowlabel;
// 			}
// 		}

// 		return -1;
// 	}
// 	// Handle vlan tag
// 	else if (ethernet->type == 0x8100)
// 	{
// 		struct dot1q_t *dot1q = cursor_advance(cursor, sizeof(*dot1q));

// 		if (dot1q->type == 0x86DD) {
// 			struct ip6_t *ip6 = cursor_advance(cursor, sizeof(*ip6));

// 			struct fourtuple addrport;

// 			// This is necessary for some reason to do with compiler padding
// 			__builtin_memset(&addrport, 0, sizeof(addrport));

// 			addrport.ip6_hi = ip6->dst_hi;
// 			addrport.ip6_lo = ip6->dst_lo;

// 			// TCP
// 			if (ip6->next_header == 6) {
// 				struct tcp_t *tcp = cursor_advance(cursor, sizeof(*tcp));

// 				addrport.dport = tcp->dst_port;
// 				addrport.sport = tcp->src_port;

// 				u64 *delete = tobedeleted.lookup(&addrport);

// 				u64 *flowlabel = flowlabel_table.lookup(&addrport);

// 				if (delete) {
// 					flowlabel_table.delete(&addrport);
// 					tobedeleted.delete(&addrport);
// 				}
// 				else if (flowlabel) {
// 					ip6->flow_label = *flowlabel;
// 				}
// 			}
// 			return -1;
// 		}
// 		else {
// 			return -1;
// 		}
// 	}
// 	else {
// 		return -1;
// 	}
// }
