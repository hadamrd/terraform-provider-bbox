terraform {
  required_providers {
    bbox = {
      source = "hadamrd/bbox"
    }
  }
}

provider "bbox" {}

data "bbox_wan" "current" {}

output "public_ip" {
  value = data.bbox_wan.current.ip_v4
}

resource "bbox_nat_rule" "ssh" {
  name          = "ssh"
  external_port = 22222
  target_ip     = "192.168.1.42"
  internal_port = 22
  protocol      = "tcp"
}
