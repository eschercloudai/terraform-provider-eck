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

resource "eck_cluster" "terraform" {
  name              = "terraform"
  eckcp             = "default"
  applicationbundle = "kubernetes-cluster-1.3.1"
  clusternetwork = {
    dnsnameservers = ["1.1.1.1", "1.0.0.1"]
    nodeprefix     = "192.168.0.0/16"
    serviceprefix  = "172.16.0.0/12"
    podprefix      = "10.0.0.0/8"
  }
  clusteropenstack = {
    externalnetworkid = "c9d130bc-301d-45c0-9328-a6964af65579"
  }
  controlplane = {
    flavor   = "g.2.standard"
    image    = "eck-230714-4bef8ab1"
    replicas = 3
    version  = "v1.27.2"
  }
  clusterfeatures = {
    autoscaling = true
  }
  workloadnodepools = [{
    name     = "cpu"
    replicas = 3
    image    = "eck-230714-4bef8ab1"
    version  = "v1.27.2"
    flavor   = "g.2.standard"
    autoscaling = {
      minimum = 1
      maximum = 2
    }
    },
    {
      name     = "gpu"
      replicas = 1
      image    = "eck-230714-4bef8ab1"
      version  = "v1.27.2"
      flavor   = "g.2.standard"
    }
  ]
}

output "cluster_config" {
  value = eck_cluster.conformance.kubeconfig
}

