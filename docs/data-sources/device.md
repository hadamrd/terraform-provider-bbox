# bbox_device (Data Source)

Router metadata — model, serial, firmware, uptime.

## Example Usage

```terraform
data "bbox_device" "router" {}

output "firmware" {
  value = data.bbox_device.router.firmware
}
```

## Attribute Reference

- `model` (String) — router model.
- `serial` (String) — serial number.
- `firmware` (String) — firmware version.
- `uptime_seconds` (Number) — seconds since last boot.
