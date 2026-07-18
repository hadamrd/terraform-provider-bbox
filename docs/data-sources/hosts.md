# bbox_hosts (Data Source)

List every LAN host the router has seen.

## Example Usage

```terraform
data "bbox_hosts" "active" {
  active_only = true
}

output "active" {
  value = [for h in data.bbox_hosts.active.hosts : h.hostname]
}
```

## Argument Reference

- `active_only` (Boolean, Optional) — filter to hosts where `active = true`.

## Attribute Reference

- `hosts` (List of Object) — each entry contains `id`, `hostname`, `ip_address`, `mac`, `link`, `active`.
