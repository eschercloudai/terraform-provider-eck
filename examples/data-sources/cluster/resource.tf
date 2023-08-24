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
  project  = "abc123"
}

data "eck_cluster" "terratest" {
  eckcp = "default"
  name  = "terratest"
}


output "example_cluster" {
  value = data.eck_cluster.terratest
}

