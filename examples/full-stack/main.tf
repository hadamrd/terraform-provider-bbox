terraform {
  required_providers {
    bbox = {
      source = "hadamrd/bbox"
    }
  }
}

provider "bbox" {}

variable "guest_wifi_passphrase" {
  type      = string
  sensitive = true
}

variable "duckdns_token" {
  type      = string
  sensitive = true
}

# --- Port-forwards -----------------------------------------------------------
resource "bbox_nat_rule" "https" {
  name          = "https"
  external_port = 8443
  target_ip     = "192.168.1.42"
  internal_port = 443
  protocol      = "tcp"
}

# --- Static DHCP -------------------------------------------------------------
resource "bbox_dhcp_reservation" "server" {
  mac        = "aa:bb:cc:dd:ee:ff"
  ip_address = "192.168.1.42"
  hostname   = "server"
}

# --- Firewall (block outbound telnet from a specific LAN host) ---------------
resource "bbox_firewall_rule" "block_iot_telnet" {
  name     = "block-iot-telnet"
  action   = "Drop"
  protocol = "tcp"
  src_ip   = "192.168.1.55"
  dst_port = "23"
}

# --- DynDNS: expose the WAN IP as home.duckdns.org ---------------------------
resource "bbox_dyndns" "duckdns" {
  provider_name = "duckdns"
  hostname      = "home.duckdns.org"
  password      = var.duckdns_token
}

# --- WiFi: rotate the 5 GHz passphrase from a variable -----------------------
resource "bbox_wifi_band" "guest_5g" {
  band       = "5"
  ssid       = "home-guest"
  passphrase = var.guest_wifi_passphrase
}
