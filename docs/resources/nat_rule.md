# bbox_nat_rule (Resource)

A NAT / port-forward rule on the Bbox. Updates go through delete-and-recreate
since the router has no PATCH endpoint.

## Example Usage

```terraform
resource "bbox_nat_rule" "ssh" {
  name          = "ssh"
  external_port = 22222
  target_ip     = "192.168.1.42"
  internal_port = 22
  protocol      = "tcp"
}
```

## Argument Reference

- `name` (String, Required) — human-readable description; the primary logical identifier.
- `external_port` (Number, Required) — WAN port to expose. Must sit inside the MAP-T range unless `skip_port_check = true`.
- `target_ip` (String, Required) — LAN IP the traffic is forwarded to.
- `internal_port` (Number, Optional) — LAN port. Defaults to `external_port`.
- `protocol` (String, Optional) — `tcp` or `udp`. Defaults to `tcp`.
- `remote_ip` (String, Optional) — restrict source IP. Empty = any.
- `skip_port_check` (Boolean, Optional) — bypass MAP-T port-range validation. Defaults to `false`.

## Attribute Reference

- `id` (Number) — router-assigned numeric rule ID.

## Import

```bash
terraform import bbox_nat_rule.ssh <numeric-id>
```

Look up the ID with `bbox nat list`.
