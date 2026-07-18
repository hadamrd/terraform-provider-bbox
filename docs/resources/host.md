# bbox_host (Resource)

Mutable metadata (hostname, block state) for a LAN host identified by MAC. The
host must have connected at least once so it exists in the router's host table.
Destroy leaves the host on the router but removes it from Terraform state.

## Example Usage

```terraform
resource "bbox_host" "iot_hub" {
  mac      = "aa:bb:cc:dd:ee:ff"
  hostname = "iot-hub"
  blocked  = false
}
```

## Argument Reference

- `mac` (String, Required) — MAC address (any case, dashes or colons).
- `hostname` (String, Optional) — friendly hostname.
- `blocked` (Boolean, Optional) — block from LAN. Defaults to `false`.

## Attribute Reference

- `id` (Number) — router-assigned host ID.

## Import

```bash
terraform import bbox_host.iot_hub <mac-address>
```
