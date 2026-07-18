terraform {
  required_providers {
    bbox = {
      source = "hadamrd/bbox"
    }
  }
}

provider "bbox" {}

# Look up MAC addresses by friendly hostname — the device must have
# connected at least once so the Bbox has it in its host table.
data "bbox_host" "nas" {
  hostname = "synology"
}

data "bbox_host" "printer" {
  hostname = "brother-hl"
}

data "bbox_host" "laptop" {
  hostname = "khalid-thinkpad"
}

resource "bbox_dhcp_reservation" "nas" {
  mac        = data.bbox_host.nas.mac
  ip_address = "192.168.1.10"
  hostname   = "nas"
}

resource "bbox_dhcp_reservation" "printer" {
  mac        = data.bbox_host.printer.mac
  ip_address = "192.168.1.20"
  hostname   = "printer"
}

resource "bbox_dhcp_reservation" "laptop" {
  mac        = data.bbox_host.laptop.mac
  ip_address = "192.168.1.30"
  hostname   = "laptop"
}
