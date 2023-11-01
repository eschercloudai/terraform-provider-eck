package provider

import (
	"context"
	"io"

	"github.com/eschercloudai/eckctl/pkg/generated"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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

func tfMapToStringMap(ctx context.Context, value basetypes.MapValue) (*map[string]string, error) {
	mapVal := map[string]string{}
	mapValue, _ := value.ToMapValue(ctx)

	for k, v := range mapValue.Elements() {
		value, _ := v.ToTerraformValue(ctx)
		var stringValue string
		err := value.As(&stringValue)
		if err != nil {
			return nil, err
		}
		mapVal[k] = stringValue
	}

	return &mapVal, nil
}

func generateKubernetesCluster(ctx context.Context, plan clusterModel) generated.KubernetesCluster {
	var dnsNameservers []string
	plan.ClusterNetwork.DnsNameservers.ElementsAs(ctx, &dnsNameservers, false)
	workloadNodePools := generateWorkloadNodePools(ctx, plan.WorkloadNodePools)
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
			Autoscaling:         plan.ClusterFeatures.Autoscaling.ValueBoolPointer(),
			Ingress:             plan.ClusterFeatures.Ingress.ValueBoolPointer(),
			FileStorage:         plan.ClusterFeatures.Longhorn.ValueBoolPointer(),
			Prometheus:          plan.ClusterFeatures.Prometheus.ValueBoolPointer(),
			KubernetesDashboard: plan.ClusterFeatures.Dashboard.ValueBoolPointer(),
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
			Dashboard:   types.BoolValue(*cluster.Features.KubernetesDashboard),
		},
		WorkloadNodePools: generateWorkloadNodePoolModel(ctx, cluster.WorkloadPools),
	}
	return clusterModel
}

func generateWorkloadNodePools(ctx context.Context, pools []workloadNodePoolModel) generated.KubernetesClusterWorkloadPools {
	var workloadNodePools generated.KubernetesClusterWorkloadPools
	for _, pool := range pools {
		workloadNodePool := generated.KubernetesClusterWorkloadPool{
			Name: pool.Name.ValueString(),
			Machine: generated.OpenstackMachinePool{
				Disk: &generated.OpenstackVolume{
					Size: int(pool.Disk.ValueInt64()),
				},
				FlavorName: pool.Flavor.ValueString(),
				ImageName:  pool.Image.ValueString(),
				Replicas:   int(pool.Replicas.ValueInt64()),
				Version:    pool.Version.ValueString(),
			},
		}
		if pool.Autoscaling != nil {
			workloadNodePool.Autoscaling = &generated.KubernetesClusterAutoscaling{
				MinimumReplicas: int(pool.Autoscaling.MinimumReplicas.ValueInt64()),
				MaximumReplicas: int(pool.Autoscaling.MaximumReplicas.ValueInt64()),
			}
		}
		if !pool.Labels.IsNull() {
			labels, _ := tfMapToStringMap(ctx, pool.Labels)
			if labels != nil && len(*labels) != 0 {
				workloadNodePool.Labels = labels
			}
		}
		workloadNodePools = append(workloadNodePools, workloadNodePool)
	}
	return workloadNodePools
}

// Render cluster workloadpool representation for Terraform state
func generateWorkloadNodePoolModel(ctx context.Context, workloadpools generated.KubernetesClusterWorkloadPools) []workloadNodePoolModel {
	var workloadPools []workloadNodePoolModel
	for _, pool := range workloadpools {
		workloadPool := workloadNodePoolModel{
			Name:     types.StringValue(pool.Name),
			Disk:     types.Int64Value(int64(pool.Machine.Disk.Size)),
			Flavor:   types.StringValue(pool.Machine.FlavorName),
			Image:    types.StringValue(pool.Machine.ImageName),
			Replicas: types.Int64Value(int64(pool.Machine.Replicas)),
			Version:  types.StringValue(pool.Machine.Version),
		}
		if pool.Autoscaling != nil {
			workloadPool.Autoscaling = &autoscalingModel{
				MinimumReplicas: types.Int64Value(int64(pool.Autoscaling.MinimumReplicas)),
				MaximumReplicas: types.Int64Value(int64(pool.Autoscaling.MaximumReplicas)),
			}
		}
		if pool.Labels != nil && len(*pool.Labels) != 0 {
			workloadPool.Labels, _ = types.MapValueFrom(ctx, types.StringType, pool.Labels)
		} else {
			workloadPool.Labels = types.MapNull(types.StringType)
		}
		workloadPools = append(workloadPools, workloadPool)
	}
	return workloadPools
}
