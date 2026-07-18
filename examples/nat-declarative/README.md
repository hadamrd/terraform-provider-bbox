# nat-declarative

Canonical "port-forwards as code" workflow: one locals map, `for_each` fan-out.

Add/remove a key in `local.port_forwards`, `terraform apply`, done. State survives
router reboots because the router-assigned rule IDs are stored in `terraform.tfstate`.

If a port falls outside the MAP-T range the router allocates to you, add
`skip_port_check = true` to that entry — Bouygues rejects it otherwise.
