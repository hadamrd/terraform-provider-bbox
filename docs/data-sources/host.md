# bbox_host (Data Source)

Look up one LAN host by any single identifier. Set exactly one of `id`,
`hostname`, `mac`, or `ip_address`.

## Example Usage

```terraform
data "bbox_host" "nas" {
  hostname = "synology"
}

output "nas_ip" {
  value = data.bbox_host.nas.ip_address
}
```

## Argument Reference

- `id` (Number, Optional) — router-assigned host ID.
- `hostname` (String, Optional) — friendly hostname.
- `ip_address` (String, Optional) — LAN IP.
- `mac` (String, Optional) — MAC address.

## Attribute Reference

- `link` (String) — `Wifi 5`, `Wifi 2.4`, `Ethernet`, or `Offline`.
- `active` (Boolean) — currently online.

Whichever identifier(s) were not supplied are populated as computed values.
