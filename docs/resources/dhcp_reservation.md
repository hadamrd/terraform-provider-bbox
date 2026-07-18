# bbox_dhcp_reservation (Resource)

A DHCP static reservation on the Bbox. Pins a fixed LAN IP to a MAC address.
The host must have connected once so the router has it in its client table.

## Example Usage

```terraform
resource "bbox_dhcp_reservation" "nas" {
  mac        = "aa:bb:cc:dd:ee:ff"
  ip_address = "192.168.1.10"
  hostname   = "nas"
}
```

## Argument Reference

- `mac` (String, Required) — MAC address (any case, dashes or colons; normalised on write).
- `ip_address` (String, Required) — IP to reserve.
- `hostname` (String, Optional) — optional friendly hostname.

## Attribute Reference

- `id` (Number) — router-assigned client ID.

## Import

```bash
terraform import bbox_dhcp_reservation.nas <mac-address>
```

Import is by MAC — the router-assigned ID is derived on the first Read.
