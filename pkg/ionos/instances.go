package ionos

import (
	"context"
	"errors"
	"fmt"
	client2 "github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/client"
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

func (i instances) AddClient(datacenterId string, token []byte) error {
	if i.clients[datacenterId] == nil {
		c, err := client2.New(datacenterId, token)
		if err != nil {
			return err
		}
		i.clients[datacenterId] = &c
	}
	return nil
}

// no caching
func (i instances) discoverNode(ctx context.Context, node *v1.Node) (*cloudprovider.InstanceMetadata, error) {
	for _, client := range i.clients {
		var err error
		var server *cloudprovider.InstanceMetadata
		providerID := GetUUIDFromNode(node)
		klog.Infof("discoverNode (datacenterId %s) %s %s", client.DatacenterId, node.Name, providerID)
		if providerID != "" {
			server, err = client.GetServer(ctx, providerID)
		} else {
			server, err = client.GetServerByName(ctx, node.Name)
		}
		if err != nil {
			return nil, errors.New(fmt.Sprintf("failed to discoverNode %v", err))
		}
		if server == nil {
			continue
		}
		return server, nil
	}
	return nil, errors.New("failed to discoverNode")
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
