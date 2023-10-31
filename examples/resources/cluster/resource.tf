terraform {
  required_providers {
    eck = {
      source = "registry.terraform.io/eschercloudai/eck"
    }
  }
}

provider "eck" {
  host     = "https://eck.startwell.eschercloud.dev"
  username = "nick"
  project  = "7612aafcbd454530b1d03d60038fe824"
}

resource "eck_cluster" "terratest" {
  name              = "terratest"
  eckcp             = "default"
  applicationbundle = "kubernetes-cluster-1.4.0"
  clusternetwork = {
    dnsnameservers = ["1.1.1.1", "1.0.0.1"]
    nodeprefix     = "172.16.0.0/16"
    serviceprefix  = "10.42.0.0/16"
    podprefix      = "10.43.0.0/16"
  }
  clusteropenstack = {
    externalnetworkid = "70bb46a1-4d43-485d-9dbc-4aa979990327"
  }
  controlplane = {
    flavor   = "m1.large"
    image    = "eck-231023-a16c4645"
    replicas = 1
    version  = "v1.28.3"
  }
  clusterfeatures = {
    autoscaling = false
  }
  workloadnodepools = [
    {
      name     = "cpu"
      replicas = 1
      image    = "eck-231023-a16c4645"
      version  = "v1.28.3"
      flavor   = "m1.large"
    },
    {
      name     = "gpu"
      replicas = 1
      image    = "eck-231023-a16c4645"
      version  = "v1.28.3"
      flavor   = "g1.medium.1xa100"
      labels = {
        gpu = "1xa100"
      }
    }
  ]
}

output "cluster_config" {
  value = eck_cluster.terratest.kubeconfig
}

