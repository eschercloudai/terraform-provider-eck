terraform {
  required_providers {
    eck = {
      source = "registry.terraform.io/eschercloudai/eck"
    }
  }
}

provider "eck" {
  host     = "https://eck.nl1.eschercloud.dev"
  username = "n.jones@eschercloud.ai"
  project  = "1be14bad764c421a804365a49c0060c0"
}

data "eck_controlplanes" "default" {}

data "eck_cluster" "terratest" {
  eckcp = "tftest"
  name  = "terratest"
}

output "example_controlplanes" {
  value = data.eck_controlplanes.default
}

output "example_clusters" {
  value = data.eck_cluster.terratest
}

