# bbox_upnp (Resource)

Singleton for the UPnP IGD toggle.

## Example Usage

```terraform
resource "bbox_upnp" "off" {
  enabled = false
}
```

## Argument Reference

- `enabled` (Boolean, Required) — UPnP IGD on/off.

## Attribute Reference

- `id` (String) — always `"singleton"`.

## Import

```bash
terraform import bbox_upnp.off singleton
```
