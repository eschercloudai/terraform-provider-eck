# Terraform Provider for the EscherCloud Kubernetes Service

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0

## Using the ECK Terraform Provider

To quickly get started with the ECK Terraform Provider, use the following example:

```tf
terraform {
  required_providers {
    eck = {
      source  = "eschercloudai/eck"
      version = "0.0.5"
    }
  }
}

provider "eck" {
  host     = "https://eck.startwell.eschercloud.dev"
  username = "openstackuser"
  password = "hunter2"
  project  = "openstackprojectid"
}

variable "k8s_version" {
  type        = string
  default     = "v1.28.3"
  description = "The version of Kubernetes to install"
}

variable "eck_image" {
  type        = string
  default     = "eck-231023-a16c4645"
  description = "The name of the ECK image to use"
}

resource "eck_cluster" "demo" {
  name = "demo"
  wait = true
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
    replicas = 1
    image    = var.eck_image
    version  = var.k8s_version
  }
  clusterfeatures = {
    autoscaling = false
    ingress     = false
  }
  workloadnodepools = [
    {
      name     = "cpu"
      replicas = 1
      image    = var.eck_image
      version  = var.k8s_version
      flavor   = "m1.large"
    },
    {
      name     = "gpu"
      replicas = 1
      image    = var.eck_image
      version  = var.k8s_version
      flavor   = "g1.medium.1xa100"
    }
  ]
}

resource "local_file" "kubeconfig" {
  content         = eck_cluster.demo.kubeconfig
  filename        = "kubeconfig"
  file_permission = 0644
}

```

Amend various options to suit the target infrastructure, notably:
* `project`: This should be the UUID of the project you want to deploy your cluster into
* `eck_image`: The name of the OS image to be used for provisioning nodes

Then, to deploy your cluster:

```
terraform init # only required the first time
terraform plan -out plan.out
terraform apply plan.out
```

After a brief period of time the cluster will be provisioned and the resulting kubeconfig written to `kubeconfig`:

```
eck_cluster.demo: Creation complete after 3m33s [name=demo]
local_file.kubeconfig: Creating...
local_file.kubeconfig: Creation complete after 0s [id=7b1ece5a85463a0ffb3e0d91ff6162a47663dd95]
% export KUBECONFIG=$(pwd)/kubeconfig
% kubectl get nodes
NAME                                            STATUS   ROLES           AGE     VERSION
cluster-a8d33e78-control-plane-5e671f5e-cl6gd   Ready    control-plane   5m33s   v1.28.3
cluster-a8d33e78-pool-68ab84f7-b5652fd1-42tst   Ready    <none>          4m53s   v1.28.3
cluster-a8d33e78-pool-68ab84f7-b5652fd1-42tst   Ready    <none>          4m53s   v1.28.3
```

For more information on ECK, consult the official [ECK documentation](https://docs.eschercloud.ai/Kubernetes/), and the Terraform Resource-specific docs are in [docs](./docs).