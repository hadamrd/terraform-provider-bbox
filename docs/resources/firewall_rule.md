# bbox_firewall_rule (Resource)

A firewall rule on the Bbox. Updates go through delete-and-recreate.

## Example Usage

```terraform
resource "bbox_firewall_rule" "block_telnet" {
  name     = "block-iot-telnet"
  action   = "Drop"
  protocol = "tcp"
  src_ip   = "192.168.1.55"
  dst_port = "23"
}
```

## Argument Reference

- `name` (String, Required) — rule description.
- `action` (String, Required) — `Drop` or `Accept`.
- `protocol` (String, Required) — one of `tcp`, `udp`, `icmp`, `esp`, `ah`, `icmpv6`, `igmp`, `gre`.
- `dst_ip` (String, Optional) — destination IP or empty.
- `dst_port` (String, Optional) — port or range, e.g. `"8000-8100"`.
- `src_ip` (String, Optional) — source IP or empty.
- `src_port` (String, Optional) — port or range.
- `enabled` (Boolean, Optional) — defaults to `true`.
- `ip_version` (String, Optional) — `IPv4` or `IPv6`. Defaults to `IPv4`.

## Attribute Reference

- `id` (Number) — router-assigned rule ID.

## Import

```bash
terraform import bbox_firewall_rule.block_telnet <numeric-id>
```
