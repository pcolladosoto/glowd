# Netlink-based context gathering
This plugin leverages the `sock_diag(7)` subsystem for gathering context on sockets with an
established TCP session. This information is then appended to fireflies as stated in the
SciTags technical specification.

## Gathered context
An example of the information gathered for socket would be:

```json
{
    "skBuff": {
        "Family": 2,
        "State": 1,
        "Timer": 2,
        "Retrans": 0,
        "ID": {
            "SourcePort": 60606,
            "DestinationPort": 443,
            "Source": "172.17.0.2",
            "Destination": "54.195.168.190",
            "Interface": 0,
            "Cookie": [
                8,
                0
            ]
        },
        "Expires": 58833,
        "RQueue": 0,
        "WQueue": 0,
        "UID": 0,
        "INode": 103936
    },
    "tcpInfo": {
        "state": 1,
        "caState": 0,
        "retransmits": 0,
        "probes": 0,
        "backoff": 0,
        "options": 5,
        "sndWscale": 7,
        "rcvdWscale": 7,
        "deliveryRateAppLimited": 0,
        "fastOpenClientFail": 0,
        "rto": 426000,
        "ato": 40000,
        "sndMss": 65483,
        "rcvMss": 6872,
        "unAcked": 0,
        "sAcked": 0,
        "lost": 0,
        "retrans": 0,
        "fAckets": 0,
        "lastDataSent": 825,
        "lastAckSent": 0,
        "lastDataRecv": 677,
        "lastAckRecv": 677,
        "pMtu": 65535,
        "rcvSsThresh": 33280,
        "rtt": 101800,
        "rttVar": 81005,
        "sndSsThresh": 2147483647,
        "sndCwnd": 10,
        "advMss": 65483,
        "reordering": 3,
        "rcvRtt": 0,
        "rcvSpace": 33280,
        "totalRetrans": 0,
        "pacingRate": 12864950,
        "maxPacingRate": 18446744073709551615,
        "bytesAcked": 746,
        "bytesRecv": 7456,
        "segsOut": 7,
        "segsIn": 7,
        "notsentBytes": 0,
        "minRtt": 494,
        "dataSegsIn": 3,
        "dataSegsOut": 3,
        "deliveryRate": 132556680,
        "busyTime": 2000,
        "rwndLimited": 0,
        "sndBufLimited": 0,
        "delivered": 4,
        "deliveredCe": 0,
        "bytesSent": 745,
        "bytesRetrans": 0,
        "dsAckDups": 0,
        "reordSeen": 0,
        "rcvOooPack": 0,
        "sndWnd": 524288
    },
    "bbr": null,
    "tos": {
        "tos": 0
    },
    "memInfo": {
        "rMem": 0,
        "wMem": 0,
        "fMem": 0,
        "tMem": 0
    },
    "skMemInfo": {
        "rMemAlloc": 0,
        "rcvBuff": 131072,
        "WMemAlloc": 0,
        "sndBuff": 2626560,
        "fwdAlloc": 0,
        "wMemQueued": 0,
        "optMem": 0,
        "backlog": 0,
        "drops": 0
    },
    "cong": {
        "algorithm": "cubic"
    },
    "vegasInfo": null,
    "dctcpInfo": null
}
```
