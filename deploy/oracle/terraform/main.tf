terraform {
  required_version = ">= 1.5.0"
  required_providers {
    oci = {
      source  = "oracle/oci"
      version = ">= 6.0.0"
    }
  }
}

provider "oci" {
  region = var.region
}

data "oci_identity_availability_domains" "available" {
  compartment_id = var.tenancy_ocid
}

data "oci_core_images" "ubuntu" {
  compartment_id           = var.compartment_ocid
  operating_system         = "Canonical Ubuntu"
  operating_system_version = "24.04"
  shape                    = "VM.Standard.A1.Flex"
  sort_by                  = "TIMECREATED"
  sort_order               = "DESC"
}

resource "oci_core_vcn" "buswatch" {
  compartment_id = var.compartment_ocid
  cidr_blocks    = ["10.42.0.0/16"]
  display_name   = "thai-bus-watch-vcn"
  dns_label      = "buswatch"
}

resource "oci_core_internet_gateway" "buswatch" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.buswatch.id
  display_name   = "thai-bus-watch-internet-gateway"
  enabled        = true
}

resource "oci_core_route_table" "public" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.buswatch.id
  display_name   = "thai-bus-watch-public-routes"

  route_rules {
    destination       = "0.0.0.0/0"
    destination_type  = "CIDR_BLOCK"
    network_entity_id = oci_core_internet_gateway.buswatch.id
  }
}

resource "oci_core_security_list" "buswatch" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.buswatch.id
  display_name   = "thai-bus-watch-firewall"

  egress_security_rules {
    protocol    = "all"
    destination = "0.0.0.0/0"
  }

  ingress_security_rules {
    protocol = "6"
    source   = var.ssh_allowed_cidr
    tcp_options {
      min = 22
      max = 22
    }
    description = "SSH administration"
  }

  ingress_security_rules {
    protocol = "6"
    source   = "0.0.0.0/0"
    tcp_options {
      min = 80
      max = 80
    }
    description = "HTTP"
  }

  ingress_security_rules {
    protocol = "6"
    source   = "0.0.0.0/0"
    tcp_options {
      min = 443
      max = 443
    }
    description = "HTTPS"
  }

  ingress_security_rules {
    protocol = "17"
    source   = "0.0.0.0/0"
    udp_options {
      min = 443
      max = 443
    }
    description = "HTTP/3"
  }
}

resource "oci_core_subnet" "public" {
  compartment_id             = var.compartment_ocid
  vcn_id                     = oci_core_vcn.buswatch.id
  cidr_block                 = "10.42.1.0/24"
  display_name               = "thai-bus-watch-public-subnet"
  dns_label                  = "public"
  prohibit_public_ip_on_vnic = false
  route_table_id             = oci_core_route_table.public.id
  security_list_ids          = [oci_core_security_list.buswatch.id]
}

resource "oci_core_instance" "buswatch" {
  availability_domain = data.oci_identity_availability_domains.available.availability_domains[var.availability_domain_index].name
  compartment_id      = var.compartment_ocid
  display_name        = "thai-bus-watch"
  shape               = "VM.Standard.A1.Flex"

  shape_config {
    ocpus         = 1
    memory_in_gbs = 6
  }

  create_vnic_details {
    subnet_id        = oci_core_subnet.public.id
    assign_public_ip = true
    display_name     = "thai-bus-watch-vnic"
    hostname_label   = "thai-bus-watch"
  }

  source_details {
    source_type             = "image"
    source_id               = data.oci_core_images.ubuntu.images[0].id
    boot_volume_size_in_gbs = 50
  }

  metadata = {
    ssh_authorized_keys = trimspace(var.ssh_public_key)
    user_data = base64encode(templatefile("${path.module}/cloud-init.yaml", {
      repository_url = var.repository_url
      branch         = var.repository_branch
    }))
  }
}

output "public_ip" {
  description = "Public IPv4 address of the Thai Bus Watch server"
  value       = oci_core_instance.buswatch.public_ip
}

output "test_url" {
  description = "Initial HTTP test URL; configure the domain deployment for HTTPS afterward"
  value       = "http://${oci_core_instance.buswatch.public_ip}"
}

output "ssh_command" {
  description = "SSH destination (add your local -i private-key path)"
  value       = "ssh ubuntu@${oci_core_instance.buswatch.public_ip}"
}
