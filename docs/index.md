# bbox Provider

Manage a Bouygues Bbox router via its reverse-engineered admin API. Declare NAT
rules, DHCP reservations, firewall rules, WiFi bands, DynDNS, and host metadata
in HCL and let Terraform reconcile them. Backed by the same client library that
powers [hadamrd/bbox-cli](https://github.com/hadamrd/bbox-cli).

## Example Usage

```terraform
terraform {
  required_providers {
    bbox = {
      source = "hadamrd/bbox"
    }
  }
}

provider "bbox" {
  # session_file  = "~/.bbox-session.json"
  # password_file = "~/.bbox-password"
  # base_url      = "https://mabbox.bytel.fr"
  # retries       = 2
  # timeout       = "15s"
}
```

Run `bbox login` once to seed `~/.bbox-session.json`, or set `BBOX_PASSWORD`
for headless / CI use.

## Schema

### Optional

- `session_file` (String) — path to cached session cookies. Default `~/.bbox-session.json`.
- `password_file` (String) — path to admin password file. Default `~/.bbox-password`.
- `base_url` (String) — router base URL. Default `https://mabbox.bytel.fr`.
- `retries` (Number) — retry count for transient network errors. Default `2`.
- `timeout` (String) — HTTP timeout as a Go duration (e.g. `"15s"`). Default `15s`.

## Environment Variables

| Variable             | Overrides                                         |
| -------------------- | ------------------------------------------------- |
| `BBOX_SESSION_FILE`  | `session_file`                                    |
| `BBOX_PASSWORD_FILE` | `password_file`                                   |
| `BBOX_PASSWORD`      | reads password directly (wins over the file)      |
| `BBOX_BASE_URL`      | `base_url`                                        |
| `BBOX_RETRIES`       | `retries`                                         |
| `BBOX_TIMEOUT`       | `timeout`                                         |

## See Also

- [bbox-cli](https://github.com/hadamrd/bbox-cli) — underlying CLI + reversed API surface.
- [`examples/`](https://github.com/hadamrd/terraform-provider-bbox/tree/main/examples) — basic, nat-declarative, dhcp-reservations, full-stack, data-audit.
