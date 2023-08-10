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

resource "eck_controlplane" "tftest" {
  name = "tftest"
  applicationbundle = {
    autoupgrade = true
    version     = "1.1.0"
  }
}

output "controlplane_creation" {
  value = eck_controlplane.tftest
}

