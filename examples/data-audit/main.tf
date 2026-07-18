terraform {
  required_providers {
    bbox = {
      source = "hadamrd/bbox"
    }
  }
}

provider "bbox" {}

# Read-only: no resources, just data sources + outputs. Run:
#   terraform apply -auto-approve && terraform output -json > audit.json

data "bbox_wan" "current" {}

data "bbox_device" "router" {}

data "bbox_hosts" "active" {
  active_only = true
}

output "router" {
  value = {
    model    = data.bbox_device.router.model
    firmware = data.bbox_device.router.firmware
    uptime_s = data.bbox_device.router.uptime_seconds
  }
}

output "wan" {
  value = {
    ipv4       = data.bbox_wan.current.ip_v4
    ipv6       = data.bbox_wan.current.ip_v6
    state      = data.bbox_wan.current.state
    port_range = data.bbox_wan.current.port_range
  }
}

output "active_hosts" {
  value = [
    for h in data.bbox_hosts.active.hosts : {
      hostname = h.hostname
      ip       = h.ip_address
      mac      = h.mac
      link     = h.link
    }
  ]
}
