# Netlink plugin
This plugin leverages the `sock_diag(7)` *Netlink* for the creation of *flow events*.

## Use
This plugin doesn't require any active user input. Sockets will be polled and information will be automatically
extracted.

## Configuration
Please refer to the Markdown-formatted documentation at the repository's root for more information on available
options. The following replicates the default configuration:

```json
{
    "plugins": {
        "netlink": {
            "pollIntervalSeconds": 5,
            "experimentID": 55,
            "activityID": 55
        }
    }
}
