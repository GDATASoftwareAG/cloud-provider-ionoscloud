package ionos

import (
	"context"
	"github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/config"
	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
	"strings"
)

var _ cloudprovider.InstancesV2 = &instances{}

func GetUUIDFromNode(node *v1.Node) string {
	if node == nil {
		return ""
	}
	withoutPrefix := strings.TrimPrefix(node.Spec.ProviderID, config.ProviderPrefix)
	return strings.ToLower(strings.TrimSpace(withoutPrefix))
}

// no caching
func (i instances) discoverNode(ctx context.Context, node *v1.Node) (*cloudprovider.InstanceMetadata, error) {
	providerID := GetUUIDFromNode(node)
	klog.Infof("discoverNode %s %s", node.Name, providerID)
	if providerID != "" {
		return i.client.GetServer(ctx, i.datacenterId, providerID)
	}
	return i.client.GetServerByName(ctx, i.datacenterId, node.Name)
}

func (i instances) InstanceExists(ctx context.Context, node *v1.Node) (bool, error) {
	klog.Infof("InstanceExists %s", node.Name)
	server, err := i.discoverNode(ctx, node)
	klog.InfoDepth(1, server)
	return server != nil, err

}

func (i instances) InstanceShutdown(_ context.Context, node *v1.Node) (bool, error) {
	klog.Infof("InstanceShutdown %s", node.Name)
	// TODO check here for mounted volumes
	return true, nil
}

func (i instances) InstanceMetadata(ctx context.Context, node *v1.Node) (*cloudprovider.InstanceMetadata, error) {
	klog.Infof("InstanceMetadata %s", node.Name)
	server, err := i.discoverNode(ctx, node)
	klog.InfoDepth(1, server)
	return server, err
}
