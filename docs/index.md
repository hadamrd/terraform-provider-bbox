# bbox Provider

Manage a Bouygues Bbox router via its reverse-engineered admin API.

## Example

```terraform
terraform {
  required_providers {
    bbox = {
      source = "hadamrd/bbox"
    }
  }
}

provider "bbox" {}
```

## Schema

### Optional

- `session_file` (String) — path to cached session cookies. Env: `BBOX_SESSION_FILE`. Default `~/.bbox-session.json`.
- `password_file` (String) — path to admin password file. Env: `BBOX_PASSWORD_FILE`. Default `~/.bbox-password`.
- `base_url` (String) — router base URL. Env: `BBOX_BASE_URL`. Default `https://mabbox.bytel.fr`.
- `retries` (Number) — retry count for transient network errors. Env: `BBOX_RETRIES`. Default `2`.
- `timeout` (String) — HTTP timeout, Go duration. Env: `BBOX_TIMEOUT`. Default `15s`.

`BBOX_PASSWORD` (env) overrides `password_file` when set.
