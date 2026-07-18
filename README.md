# terraform-provider-bbox

Declarative management of the Bouygues Bbox router admin API via Terraform. Backed by the reverse-engineered surface in [hadamrd/bbox-cli](https://github.com/hadamrd/bbox-cli).

Status: skeleton. No resources / data sources shipped yet (see roadmap below).

## Install (dev-override)

Until the provider is published to the Terraform Registry, run against a local build:

```powershell
go build -o $env:USERPROFILE\.terraform.d\plugins\terraform-provider-bbox.exe .
```

Then add to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "hadamrd/bbox" = "C:/Users/you/.terraform.d/plugins"
  }
  direct {}
}
```

## Example

```hcl
terraform {
  required_providers {
    bbox = {
      source = "hadamrd/bbox"
    }
  }
}

provider "bbox" {
  # session_file  = "~/.bbox-session.json"  # optional
  # password_file = "~/.bbox-password"      # optional
  # base_url      = "https://mabbox.bytel.fr"
  # retries       = 2
  # timeout       = "15s"
}

# (coming in v0.1)
# data "bbox_wan_ip" "current" {}
#
# resource "bbox_nat_rule" "retrobot" {
#   name     = "retrobot"
#   protocol = "tcp"
#   external = 5555
#   internal = 5555
#   host_ip  = "192.168.1.42"
# }
```

## Configuration

| Attribute       | Env                   | Default                      | Description                       |
| --------------- | --------------------- | ---------------------------- | --------------------------------- |
| `session_file`  | `BBOX_SESSION_FILE`   | `~/.bbox-session.json`       | Cached login cookies.             |
| `password_file` | `BBOX_PASSWORD_FILE`  | `~/.bbox-password`           | Admin password file.              |
| `base_url`      | `BBOX_BASE_URL`       | `https://mabbox.bytel.fr`    | Router admin origin.              |
| `retries`       | `BBOX_RETRIES`        | `2`                          | Transient network retries.        |
| `timeout`       | `BBOX_TIMEOUT`        | `15s`                        | HTTP timeout (Go duration).       |

`BBOX_PASSWORD` (env) takes precedence over `password_file`.

## Development

```powershell
go mod tidy
go build ./...
go test ./...
```

## Roadmap

- v0.1 — `bbox_nat_rule`, `bbox_dhcp_reservation`, `bbox_wifi_ssid`, `data.bbox_wan_ip`.
- v0.2 — DMZ, UPnP, firewall, DynDNS.
- v0.3 — full state export as a data source.

## Contributing

- Follow the code style of `bbox-cli`.
- No new abstractions until a second resource needs them.
- Every resource ships with an acceptance test that hits a fake HTTPS server (see bbox-cli's `httptest` pattern).

## License

MIT — see [LICENSE](LICENSE).
