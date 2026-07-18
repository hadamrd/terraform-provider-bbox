terraform {
  required_providers {
    bbox = {
      source = "hadamrd/bbox"
    }
  }
}

provider "bbox" {}

locals {
  # Commit this map to git — it becomes the source of truth
  # for every port-forward on the home router.
  port_forwards = {
    ssh = {
      external_port = 22222
      target_ip     = "192.168.1.42"
      internal_port = 22
      protocol      = "tcp"
    }
    http = {
      external_port = 8080
      target_ip     = "192.168.1.42"
      internal_port = 80
      protocol      = "tcp"
    }
    https = {
      external_port = 8443
      target_ip     = "192.168.1.42"
      internal_port = 443
      protocol      = "tcp"
    }
    minecraft = {
      external_port = 25565
      target_ip     = "192.168.1.55"
      internal_port = 25565
      protocol      = "tcp"
    }
  }
}

resource "bbox_nat_rule" "forwards" {
  for_each = local.port_forwards

  name          = each.key
  external_port = each.value.external_port
  target_ip     = each.value.target_ip
  internal_port = each.value.internal_port
  protocol      = each.value.protocol
}
