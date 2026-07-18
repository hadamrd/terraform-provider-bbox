# bbox_wifi_band (Resource)

WiFi settings for one radio band. Singleton per band — `destroy` is a no-op
because the radios are permanent router features. To turn a radio off, set
`enabled = false`.

## Example Usage

```terraform
resource "bbox_wifi_band" "guest_5g" {
  band       = "5"
  ssid       = "home-guest"
  passphrase = var.guest_passphrase
}
```

## Argument Reference

- `band` (String, Required) — `24`, `5`, or `6`. Immutable — changing forces replacement.
- `ssid` (String, Optional) — network name.
- `passphrase` (String, Optional, Sensitive) — WPA passphrase.
- `channel` (Number, Optional) — channel number. `0` = auto.
- `enabled` (Boolean, Optional) — radio on/off.

## Attribute Reference

- `id` (String) — same as `band`.

## Import

```bash
terraform import bbox_wifi_band.guest_5g 5
```

Import key is the `band` string (`24`, `5`, or `6`).
