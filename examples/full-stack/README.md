# full-stack

Kitchen-sink example touching every mutating resource: NAT, DHCP, firewall,
DynDNS, WiFi passphrase rotation.

Pass sensitive inputs at the CLI or via env:

```bash
export TF_VAR_guest_wifi_passphrase='...'
export TF_VAR_duckdns_token='...'
terraform apply
```
