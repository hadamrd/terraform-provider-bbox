# terraform-provider-bbox

Declaratively manage a Bouygues Bbox router via its reverse-engineered admin API — NAT rules, DHCP reservations, firewall rules, WiFi bands, DynDNS, host metadata, UPnP.

[![CI](https://img.shields.io/github/actions/workflow/status/hadamrd/terraform-provider-bbox/ci.yml?branch=main&label=ci)](https://github.com/hadamrd/terraform-provider-bbox/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/hadamrd/terraform-provider-bbox?include_prereleases&sort=semver)](https://github.com/hadamrd/terraform-provider-bbox/releases)
[![License](https://img.shields.io/github/license/hadamrd/terraform-provider-bbox)](LICENSE)

Backed by [hadamrd/bbox-cli](https://github.com/hadamrd/bbox-cli), so authentication and session caching behave identically to the CLI.

## Quick start

1. Build and install the provider into the local plugin cache:

   ```bash
   make install
   ```

2. Add a `dev_overrides` block to `~/.terraformrc` (until the provider ships to the Registry):

   ```hcl
   provider_installation {
     dev_overrides {
       "hadamrd/bbox" = "/home/you/.terraform.d/plugins/registry.terraform.io/hadamrd/bbox/0.1.0/linux_amd64"
     }
     direct {}
   }
   ```

3. Drop a `main.tf`:

   ```hcl
   terraform {
     required_providers {
       bbox = { source = "hadamrd/bbox" }
     }
   }

   provider "bbox" {}

   data "bbox_wan" "current" {}

   resource "bbox_nat_rule" "ssh" {
     name          = "ssh"
     external_port = 22222
     target_ip     = "192.168.1.42"
     internal_port = 22
     protocol      = "tcp"
   }
   ```

4. Seed the session and plan:

   ```bash
   bbox login                 # or: export BBOX_PASSWORD=...
   terraform plan
   ```

## Feature matrix

### Resources

| Resource                  | Endpoint                     | Notes                                             |
| ------------------------- | ---------------------------- | ------------------------------------------------- |
| `bbox_nat_rule`           | `/api/v1/nat/rules`          | Port-forward. Delete+recreate on update.          |
| `bbox_dhcp_reservation`   | `/api/v1/dhcp/clients`       | Fixed IP for a MAC. Host must be known.           |
| `bbox_firewall_rule`      | `/api/v1/firewall/rules`     | Drop/Accept for IPv4 or IPv6.                     |
| `bbox_wifi_band`          | `/api/v1/wireless/*`         | Singleton per band (`24`/`5`/`6`).                |
| `bbox_dyndns`             | `/api/v1/dyndns`             | Singleton. DuckDNS/no-ip/ovh/etc.                 |
| `bbox_host`               | `/api/v1/hosts/{id}`         | Rename or block a known LAN host.                 |
| `bbox_upnp`               | `/api/v1/upnp/igd`           | Singleton toggle.                                 |
| `bbox_wifi_acl`           | `/api/v1/wireless/acl`       | Singleton toggle for MAC filtering. Enabling can lock out WiFi. |
| `bbox_wifi_acl_rule`      | `/api/v1/wireless/acl/rules` | A MAC access-control entry.                       |
| `bbox_wifi_schedule`      | `/api/v1/wireless/scheduler` | A recurring WiFi-pause window (radios off).       |
| `bbox_parental_control`   | `/api/v1/parentalcontrol`    | Singleton: enable + default policy.               |
| `bbox_parental_rule`      | `/api/v1/parentalcontrol/scheduler` | A parental-control access window.          |

```hcl
# WiFi off on weeknights + block a MAC + weekend-only kid access
resource "bbox_wifi_schedule" "night" {
  name       = "School nights"
  days       = ["mon", "tue", "wed", "thu", "fri"]
  start_time = "23:30"
  end_time   = "06:30"
}

resource "bbox_wifi_acl_rule" "blocked" {
  macaddress = "aa:bb:cc:dd:ee:ff"
}

resource "bbox_parental_control" "pc" {
  enabled        = true
  default_policy = "Forbidden" # windows grant access
}

resource "bbox_parental_rule" "weekend" {
  name       = "Weekend screen time"
  days       = ["sat", "sun"]
  start_time = "09:00"
  end_time   = "20:00"
}
```

### Data sources

| Data source     | Endpoint             | Notes                                |
| --------------- | -------------------- | ------------------------------------ |
| `bbox_wan`      | `/api/v1/wan/ip`     | WAN IP + MAP-T port range.           |
| `bbox_host`     | `/api/v1/hosts`      | Resolve by id/hostname/mac/ip.       |
| `bbox_hosts`    | `/api/v1/hosts`      | List all hosts.                      |
| `bbox_device`   | `/api/v1/device`     | Model, serial, firmware, uptime.     |

## Documentation

- Registry-style docs: [`docs/`](docs/) — one page per resource and data source.
- End-to-end scenarios: [`examples/`](examples/) — `basic`, `nat-declarative`, `dhcp-reservations`, `full-stack`, `data-audit`.
- Provider config: [`docs/index.md`](docs/index.md).

## Development

Requirements: Go 1.22+, Terraform 1.5+.

```bash
make build       # ./bin/terraform-provider-bbox
make install     # copies binary into ~/.terraform.d/plugins/registry.terraform.io/hadamrd/bbox/0.1.0/<os>_<arch>/
make test        # go test ./... -race -count=1
make vet
make fmt         # gofmt -w . && terraform fmt -recursive examples/
make testacc     # acceptance tests — require a live Bbox router
```

## Contributing

- Follow the code style of [bbox-cli](https://github.com/hadamrd/bbox-cli).
- No new abstractions until a second consumer demands them.
- Every resource ships with a unit test hitting a fake HTTPS server (see the `bbox-cli` `httptest` pattern).
- Add a `CHANGELOG.md` entry under `[Unreleased]` for every user-facing change.

## License

MIT — see [LICENSE](LICENSE).
