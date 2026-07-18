# Changelog

All notable changes to this project are documented here. Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versions follow SemVer.

## [Unreleased] — targeting v0.1.0

### Added
- `examples/` directory with five end-to-end scenarios: `basic`, `nat-declarative`, `dhcp-reservations`, `full-stack`, `data-audit`.
- Registry-style docs under `docs/resources/` and `docs/data-sources/` for every published resource and data source.
- `GNUmakefile` with `build`, `install`, `test`, `testacc`, `fmt`, `vet`, `lint`, `clean` targets. `install` drops the binary into the local Terraform plugin cache under `~/.terraform.d/plugins/registry.terraform.io/hadamrd/bbox/0.1.0/<os>_<arch>/`.
- CI validates every `examples/*/main.tf` with `terraform fmt -check` and `terraform validate`.
- Root `CHANGELOG.md`.

### Changed
- README rewritten as a proper front-door with badges, quick-start, feature matrix, and dev-install flow.

## [v0.1.0-beta] — Phase 4

### Added
- Resources: `bbox_dyndns`, `bbox_host`, `bbox_upnp`.
- Data sources: `bbox_host`, `bbox_hosts`, `bbox_device`.

## [v0.1.0-alpha] — Phase 3

### Added
- Provider skeleton (framework-based) wired to `bbox-cli`'s `pkg/client`.
- Resources: `bbox_nat_rule`, `bbox_dhcp_reservation`, `bbox_firewall_rule`, `bbox_wifi_band`.
- Data source: `bbox_wan`.
- Configuration attributes: `session_file`, `password_file`, `base_url`, `retries`, `timeout` with env-var overrides.
