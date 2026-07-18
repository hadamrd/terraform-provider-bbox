# basic

Smallest working example: read the WAN state and declare one port-forward.

## Dev-override

Until the provider is on the Registry, add to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "hadamrd/bbox" = "/path/to/repo/bin"
  }
  direct {}
}
```

Then:

```bash
make -C ../.. install
terraform plan
```
