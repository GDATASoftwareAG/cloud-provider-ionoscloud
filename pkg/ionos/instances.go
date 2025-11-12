package ionos

import (
    "context"
    "errors"
    "fmt"
    "strings"

    v1 "k8s.io/api/core/v1"
    cloudprovider "k8s.io/cloud-provider"
    "k8s.io/klog/v2"

    client2 "github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/client"
    "github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/config"
)

var _ cloudprovider.InstancesV2 = &instances{}

func GetUUIDFromNode(node *v1.Node) string {
    if node == nil {
        return ""
    }
    return ProviderIDWitPrefix(node.Spec.ProviderID)
}

func ProviderIDWitPrefix(providerID string) string {
    withoutPrefix := strings.TrimPrefix(providerID, config.ProviderPrefix)
    return strings.ToLower(strings.TrimSpace(withoutPrefix))
}

func (i instances) AddClient(datacenterId string, token []byte) error {
    if i.ionosClients[datacenterId] == nil {
        c, err := client2.New(datacenterId, token)
        if err != nil {
            return err
        }
        i.ionosClients[datacenterId] = &c
    }
    return nil
}

// no caching
func (i instances) discoverNode(ctx context.Context, node *v1.Node) (*cloudprovider.InstanceMetadata, *client2.IONOSClient, error) {
    for _, client := range i.ionosClients {
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
            return nil, nil, fmt.Errorf("failed to discoverNode %v", err)
        }
        if server == nil {
            continue
        }
        return server, client, nil
    }
    return nil, nil, nil
}

func (i instances) InstanceExists(ctx context.Context, node *v1.Node) (bool, error) {
    klog.Infof("InstanceExists %s", node.Name)
    server, _, err := i.discoverNode(ctx, node)
    klog.InfoDepth(1, server)
    return server != nil, err
}

func (i instances) InstanceShutdown(ctx context.Context, node *v1.Node) (bool, error) {
    klog.Infof("InstanceShutdown %s", node.Name)
    server, client, err := i.discoverNode(ctx, node)
    if (server == nil && err == nil) || server == nil {
        return true, nil
    }

    volumes, err := client.GetServerVolumes(ctx, ProviderIDWitPrefix(server.ProviderID))
    if err != nil {
        return false, err
    }

    for _, volume := range volumes {
        if strings.HasPrefix(volume.Name, "csi-pv.k8s.") {
            return false, nil
        }
    }
    return true, nil
}

func (i instances) InstanceMetadata(ctx context.Context, node *v1.Node) (*cloudprovider.InstanceMetadata, error) {
    klog.Infof("InstanceMetadata %s", node.Name)
    server, _, err := i.discoverNode(ctx, node)
    if server == nil && err == nil {
        return nil, errors.New("failed to discoverNode")
    }
    klog.InfoDepth(1, server)
    return server, err
}
