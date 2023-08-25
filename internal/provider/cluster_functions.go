package provider

import (
	"context"
	"io"

	"github.com/eschercloudai/eckctl/pkg/generated"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func getKubeconfig(client generated.ClientWithResponses, ctx context.Context, eckcp string, cluster string) string {
	k, err := client.GetApiV1ControlplanesControlPlaneNameClustersClusterNameKubeconfig(ctx, eckcp, cluster)
	if err != nil {
		return ""
	}
	kc, err := io.ReadAll(k.Body)
	if err != nil {
		return ""
	}
	return string(kc)
}

func generateKubernetesCluster(ctx context.Context, plan clusterModel) generated.KubernetesCluster {
	var dnsNameservers []string
	plan.ClusterNetwork.DnsNameservers.ElementsAs(ctx, &dnsNameservers, false)
	workloadNodePools := generateWorkloadNodePools(plan.WorkloadNodePools)
	cluster := generated.KubernetesCluster{
		Name: plan.Name.ValueString(),
		Status: &generated.KubernetesResourceStatus{
			Status: plan.Status.ValueString(),
		},
		ApplicationBundle: generated.ApplicationBundle{
			Name:    plan.ApplicationBundle.ValueString(),
			Version: plan.ApplicationBundle.ValueString(),
		},
		ControlPlane: generated.OpenstackMachinePool{
			ImageName:  plan.ControlPlane.Image.ValueString(),
			FlavorName: plan.ControlPlane.Flavor.ValueString(),
			Replicas:   int(plan.ControlPlane.Replicas.ValueInt64()),
			Version:    plan.ControlPlane.Version.ValueString(),
		},
		Network: generated.KubernetesClusterNetwork{
			DnsNameservers: dnsNameservers,
			NodePrefix:     plan.ClusterNetwork.NodePrefix.ValueString(),
			ServicePrefix:  plan.ClusterNetwork.ServicePrefix.ValueString(),
			PodPrefix:      plan.ClusterNetwork.PodPrefix.ValueString(),
		},
		Openstack: generated.KubernetesClusterOpenStack{
			ExternalNetworkID:       plan.ClusterOpenstack.ExternalNetworkID.ValueString(),
			ComputeAvailabilityZone: plan.ClusterOpenstack.ComputeAvailabilityZone.ValueString(),
			VolumeAvailabilityZone:  plan.ClusterOpenstack.VolumeAvailabilityZone.ValueString(),
			SshKeyName:              plan.ClusterOpenstack.SshKeyName.ValueStringPointer(),
		},
		Features: &generated.KubernetesClusterFeatures{
			Autoscaling: plan.ClusterFeatures.Autoscaling.ValueBoolPointer(),
			Ingress:     plan.ClusterFeatures.Ingress.ValueBoolPointer(),
			FileStorage: plan.ClusterFeatures.Longhorn.ValueBoolPointer(),
			Prometheus:  plan.ClusterFeatures.Prometheus.ValueBoolPointer(),
		},
		WorkloadPools: workloadNodePools,
	}

	return cluster

}

func generateClusterModel(ctx context.Context, cluster generated.KubernetesCluster, eckcp string, kubeconfig string) clusterModel {
	ns, _ := types.ListValueFrom(ctx, types.StringType, cluster.Network.DnsNameservers)
	clusterModel := clusterModel{
		Name:              types.StringValue(cluster.Name),
		ApplicationBundle: types.StringValue(cluster.ApplicationBundle.Name),
		Status:            types.StringValue(cluster.Status.Status),
		EckCp:             types.StringValue(eckcp),
		Kubeconfig:        types.StringValue(kubeconfig),
		ControlPlane: &controlPlaneNodesModel{
			Flavor:   types.StringValue(cluster.ControlPlane.FlavorName),
			Image:    types.StringValue(cluster.ControlPlane.ImageName),
			Replicas: types.Int64Value(int64(cluster.ControlPlane.Replicas)),
			Version:  types.StringValue(cluster.ControlPlane.Version),
		},
		ClusterNetwork: &clusterNetworkModel{
			DnsNameservers: ns,
			NodePrefix:     types.StringValue(cluster.Network.NodePrefix),
			PodPrefix:      types.StringValue(cluster.Network.PodPrefix),
			ServicePrefix:  types.StringValue(cluster.Network.ServicePrefix),
		},
		ClusterOpenstack: &clusterOpenstackModel{
			ComputeAvailabilityZone: types.StringValue(cluster.Openstack.ComputeAvailabilityZone),
			VolumeAvailabilityZone:  types.StringValue(cluster.Openstack.VolumeAvailabilityZone),
			ExternalNetworkID:       types.StringValue(cluster.Openstack.ExternalNetworkID),
			SshKeyName:              types.StringPointerValue(cluster.Openstack.SshKeyName),
		},
		ClusterFeatures: &clusterFeaturesModel{
			Autoscaling: types.BoolValue(*cluster.Features.Autoscaling),
			Longhorn:    types.BoolValue(*cluster.Features.FileStorage),
			Ingress:     types.BoolValue(*cluster.Features.Ingress),
			Prometheus:  types.BoolValue(*cluster.Features.Prometheus),
		},
		WorkloadNodePools: generateWorkloadNodePoolModel(cluster.WorkloadPools),
	}
	return clusterModel
}

func generateWorkloadNodePools(pools []workloadNodePoolModel) generated.KubernetesClusterWorkloadPools {
	var workloadNodePools generated.KubernetesClusterWorkloadPools
	for _, pool := range pools {
		if pool.Autoscaling != nil {
			workloadNodePools = append(workloadNodePools, generated.KubernetesClusterWorkloadPool{
				Name: pool.Name.ValueString(),
				Machine: generated.OpenstackMachinePool{
					Replicas:   int(pool.Replicas.ValueInt64()),
					FlavorName: pool.Flavor.ValueString(),
					ImageName:  pool.Image.ValueString(),
					Version:    pool.Version.ValueString(),
				},
				Autoscaling: &generated.KubernetesClusterAutoscaling{
					MinimumReplicas: int(pool.Autoscaling.MinimumReplicas.ValueInt64()),
					MaximumReplicas: int(pool.Autoscaling.MaximumReplicas.ValueInt64()),
				},
			})
		} else {
			workloadNodePools = append(workloadNodePools, generated.KubernetesClusterWorkloadPool{
				Name: pool.Name.ValueString(),
				Machine: generated.OpenstackMachinePool{
					Replicas:   int(pool.Replicas.ValueInt64()),
					FlavorName: pool.Flavor.ValueString(),
					ImageName:  pool.Image.ValueString(),
					Version:    pool.Version.ValueString(),
				},
			})
		}
	}
	return workloadNodePools
}

func generateWorkloadNodePoolModel(workloadpools generated.KubernetesClusterWorkloadPools) []workloadNodePoolModel {
	var workloadPools []workloadNodePoolModel
	for _, pool := range workloadpools {
		if pool.Autoscaling != nil {
			workloadPools = append(workloadPools, workloadNodePoolModel{
				Name:     types.StringValue(pool.Name),
				Flavor:   types.StringValue(pool.Machine.FlavorName),
				Image:    types.StringValue(pool.Machine.ImageName),
				Replicas: types.Int64Value(int64(pool.Machine.Replicas)),
				Version:  types.StringValue(pool.Machine.Version),
				Autoscaling: &autoscalingModel{
					MinimumReplicas: types.Int64Value(int64(pool.Autoscaling.MinimumReplicas)),
					MaximumReplicas: types.Int64Value(int64(pool.Autoscaling.MaximumReplicas)),
				},
			})
		} else {
			workloadPools = append(workloadPools, workloadNodePoolModel{
				Name:     types.StringValue(pool.Name),
				Flavor:   types.StringValue(pool.Machine.FlavorName),
				Image:    types.StringValue(pool.Machine.ImageName),
				Replicas: types.Int64Value(int64(pool.Machine.Replicas)),
				Version:  types.StringValue(pool.Machine.Version),
			})
		}
	}
	return workloadPools
}
