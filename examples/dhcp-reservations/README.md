# dhcp-reservations

Pin fixed IPs to named devices without memorising MAC addresses. `bbox_host`
resolves the MAC from the router's known-hosts table so you can refer to a
device by its friendly hostname.

Prerequisite: the device must have connected to the LAN at least once — the
Bbox only surfaces hosts it has seen.
