# bbox_dyndns (Resource)

Singleton config for the router's DynDNS service. Only one DynDNS binding is
supported by the firmware. Destroying the resource calls `DynDNSDisable`.

## Example Usage

```terraform
resource "bbox_dyndns" "duckdns" {
  provider_name = "duckdns"
  hostname      = "home.duckdns.org"
  password      = var.duckdns_token
}
```

## Argument Reference

- `provider_name` (String, Required) — one of `duckdns`, `dyndns`, `no-ip`, `ovh`, `duiadns`, `changeip`.
- `hostname` (String, Required) — DNS name to update.
- `password` (String, Required, Sensitive) — for DuckDNS this is the API token.
- `username` (String, Optional) — leave empty for DuckDNS.
- `enabled` (Boolean, Optional) — defaults to `true`.

## Attribute Reference

- `id` (String) — always `"singleton"`.

## Import

```bash
terraform import bbox_dyndns.duckdns singleton
```
