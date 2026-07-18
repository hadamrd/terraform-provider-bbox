# bbox_wan (Data Source)

Read the current WAN state — public IP, MAP-T range, link state.

## Example Usage

```terraform
data "bbox_wan" "current" {}

output "public_ip" {
  value = data.bbox_wan.current.ip_v4
}
```

## Attribute Reference

- `ip_v4` (String) — public IPv4.
- `ip_v6` (String) — first public IPv6 or empty.
- `state` (String) — `Up` or `Down`.
- `mac` (String) — WAN interface MAC.
- `port_range` (String) — MAP-T port range, e.g. `"40960:49151"`.
- `port_range_low` (Number) — parsed low bound (`0` = full).
- `port_range_high` (Number) — parsed high bound (`0` = full).
- `map_t_enabled` (Boolean) — MAP-T on/off.
