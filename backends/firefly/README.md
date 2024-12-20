# Firefly backend
This backend will send a UDP datagram with a JSON-formatted payload in response to *flow events*.

The format of the JSON payload is defined by the following schema obtained from the [SciTags Technical Specification](scitags-spec):

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://scitags.org/schemas/v1.0.0/firefly.schema.json",
  "title": "firefly message version 1",
  "description": "A message containing metadata about a data transfer flow",
  "type": "object",

  "properties": {
    "version": {
      "description": "The version number of the message format",
      "type": "integer",
      "minimum": 1,
      "maximum": 1,
    },
    "flow-lifecycle": {
      "description": "Details about the start, end and current state of the flow",
      "type": "object",
      "properties": {
        "state": {
          "description": "The state of the flow at the moment the message was sent",
          "type": "string",
          "enum": [ "start", "end", "ongoing" ],
        },
        "start-time": {
          "description": "The UTC date/time that the flow started",
          "type": "string",
          "format": "date-time",
        },
        "end-time": {
          "description": "The UTC date/time that the flow ended",
          "type": "string",
          "format": "date-time",
        },
        "current-time": {
          "description": "The current UTC date/time that this message was launched",
          "type": "string",
          "format": "date-time",
        },
      },
      "if": { "properties": { "state": { "const": "end" } } },
      "then": {
        "required": [ "end-time" ]
      },
      "required": [ "state", "start-time" ],
      "additionalProperties": false,
    },
    "flow-id": {
      "description": "The IP 5-tuple of the original flow that this message is reporting about",
      "type": "object",
      "properties": {
        "afi": {
          "description": "The address family IPv4 or IPv6 of the original flow",
          "type": "string",
          "enum": [ "ipv4", "ipv6" ],
        },
        "src-ip": {
          "description": "The IPv4 or IPv6 source address of the original flow",
          "type": "string",
        },
        "dst-ip": {
          "description": "The IPv4 or IPv6 destination address of the original flow",
          "type": "string",
        },
        "protocol": {
          "description": "The protocol of the original flow being reported on",
          "type": "string",
          "enum": [ "tcp", "udp" ],
        },
        "src-port": {
          "description": "The layer 4 source port number of the original flow",
          "type": "integer",
          "minimum": 1,
          "maximum": 65535,
        },
        "dst-port": {
          "description": "The layer 4 destination port number of the original flow",
          "type": "integer",
          "minimum": 1,
          "maximum": 65535,
        },
      },
      "oneOf": [
        {
          "if": { "properties": { "afi": { "const": "ipv4" } } },
          "then": {
            "properties": {
              "src-ip": { "format": "ipv4" },
              "dst-ip": { "format": "ipv4" },
            },
          },
          "else": false,
        },
        {
          "if": { "properties": { "afi": { "const": "ipv6" } } },
          "then": {
            "properties": {
              "src-ip": { "format": "ipv6" },
              "dst-ip": { "format": "ipv6" },
            }
          },
          "else": false,
        },
      ],
      "required": ["afi", "src-ip", "dst-ip", "protocol", "src-port", "dst-port"],
      "additionalProperties": false,
    },
    "usage": {
      "description": "Bytes sent/received in the original flow (optional)",
      "type": "object",
      "properties": {
        "received": {
          "description": "Bytes received in the original flow",
          "type": "integer",
        },
        "sent": {
          "description": "Bytes sent in the original flow",
          "type": "integer",
        },
      },
      "additionalProperties": false,
    },
    "netlink": {
      "description": "Netlink information related to the original flow (optional)",
      "type": "object",      
    },
    "context": {
      "description": "Additional contextual information about the original flow",
      "type": "object",
      "properties": {
        "experiment-id": {
          "description": "The experiment ID that the original flow is related to",
          "type": "integer",
        },
        "activity-id": {
          "description": "The activity ID that the original flow is related to",
          "type": "integer",
        },
        "application": {
          "description": "Name and version number of the application which is initiating the original flow",
          "type": "string",
        }
      },
      "required": [ "experiment-id", "activity-id" ],
      "additionalProperties": false,
    },
  },
  "required": [ "version", "flow-lifecycle", "flow-id", "context" ],
}
```

Work is ongoing to define the internal Go `struct` based on this schema instead of manually.

Aside from the JSON object, a Syslog-like header is prepended in the datagram's payload, an example being (also extracted
from the specification):

    <134>1 2021-09-22T11:12:27.808092+00:00 26799cfec63a flowd-go - firefly-json -

At the moment, these fireflies are sent to the destination IPv{4,6} address specified in the flow event, but that can very
easily be altered.

## Configuration
Please refer to the Markdown-formatted documentation at the repository's root for more information on available
options. The following replicates the default configuration:

```json
{
    "backends": {
        "np": {
            "fireflyDestinationPort": 10514,
            "prependSyslog": false
        }
    }
}
```

<!-- REFs -->
[scitags-spec]: https://docs.google.com/document/d/1x9JsZ7iTj44Ta06IHdkwpv5Q2u4U2QGLWnUeN2Zf5ts/edit?tab=t.0
