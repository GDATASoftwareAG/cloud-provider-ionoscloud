package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/config"
	v1 "k8s.io/api/core/v1"
	"strings"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
)

type IONOSClient struct {
	client                  *ionoscloud.APIClient
	cacheDatacenterLocation map[string]string
}

func (a *IONOSClient) Initialize(token string) {
	cfg := ionoscloud.NewConfiguration("", "", token, "https://api.ionos.com/cloudapi/v6")
	a.client = ionoscloud.NewAPIClient(cfg)
	a.cacheDatacenterLocation = map[string]string{}
}

func (a *IONOSClient) GetServer(ctx context.Context, datacenterId, providerID string) (*cloudprovider.InstanceMetadata, error) {
	if a.client == nil {
		return nil, errors.New("client isn't initialized")
	}
	serverReq := a.client.ServersApi.DatacentersServersFindById(ctx, datacenterId, providerID)
	server, req, err := serverReq.Depth(1).Execute()
	if req.StatusCode == 404 {
		return nil, err
	}
	return a.convertServerToInstanceMetadata(ctx, datacenterId, &server)
}

func (a *IONOSClient) datacenterLocation(ctx context.Context, datacenterId string) (string, error) {
	if a.client == nil {
		return "", errors.New("client isn't initialized")
	}
	location, exists := a.cacheDatacenterLocation[datacenterId]
	if exists {
		return location, nil
	}
	datacenter, req, err := a.client.DataCentersApi.DatacentersFindById(ctx, datacenterId).Depth(1).Execute()
	if req.StatusCode == 404 || err != nil {
		return "", err
	}
	a.cacheDatacenterLocation[datacenterId] = *datacenter.Properties.Location
	return *datacenter.Properties.Location, nil
}

func (a *IONOSClient) convertServerToInstanceMetadata(ctx context.Context, datacenterId string, server *ionoscloud.Server) (*cloudprovider.InstanceMetadata, error) {
	if a.client == nil {
		return nil, errors.New("client isn't initialized")
	}
	location, err := a.datacenterLocation(ctx, datacenterId)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, nil
	}
	metadata := &cloudprovider.InstanceMetadata{
		ProviderID:    fmt.Sprintf("%s%s", config.ProviderPrefix, *server.Id),
		InstanceType:  fmt.Sprintf("dedicated-core-server.cpu-%s-%d.mem-%dmb", *server.Properties.CpuFamily, *server.Properties.Cores, *server.Properties.Ram),
		NodeAddresses: []v1.NodeAddress{},
		Zone:          *server.Properties.AvailabilityZone,
		Region:        strings.Replace(location, "/", "-", 1),
	}

	klog.InfoDepth(1, metadata)
	return metadata, err
}

func (a *IONOSClient) GetServerByName(ctx context.Context, datacenterId, name string) (*cloudprovider.InstanceMetadata, error) {
	klog.Infof("GetServerByName %s", name)
	if a.client == nil {
		return nil, errors.New("client is initialized")
	}
	serverReq := a.client.ServersApi.DatacentersServersGet(ctx, datacenterId)
	servers, req, err := serverReq.Depth(2).Execute()
	if req.StatusCode == 404 {
		return nil, err
	}
	if servers.Items == nil {
		return nil, nil
	}
	items := *servers.Items
	for i := range items {
		server := &items[i]
		klog.Infof("%v", server)
		klog.Infof("%v", server.Properties)
		klog.Infof("%S", *server.Properties.Name)
		if server.Properties.Name != nil && *server.Properties.Name == name {
			return a.convertServerToInstanceMetadata(ctx, datacenterId, server)
		}
	}
	return nil, err
}
