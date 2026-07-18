# data-audit

Read-only introspection: dump WAN state, device metadata, and every active LAN
host as terraform outputs. No resources — safe to run against production.

```bash
terraform apply -auto-approve
terraform output -json > router-audit.json
```
