package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/config"
	v1 "k8s.io/api/core/v1"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
)

type IONOSClient struct {
	client        *ionoscloud.APIClient
	cacheLocation string
	DatacenterId  string
}

type userpassword struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func New(datacenterId string, secret []byte) (IONOSClient, error) {
	var cfg *ionoscloud.Configuration
	if secret[0] == '{' {
		var up userpassword
		if err := json.Unmarshal(secret, &up); err != nil {
			return IONOSClient{}, err
		}
		cfg = ionoscloud.NewConfiguration(up.Username, up.Password, "", "https://api.ionos.com/cloudapi/v6")
	} else {
		cfg = ionoscloud.NewConfiguration("", "", string(secret), "https://api.ionos.com/cloudapi/v6")
	}
	a := IONOSClient{}
	a.client = ionoscloud.NewAPIClient(cfg)
	a.cacheLocation = ""
	a.DatacenterId = datacenterId
	return a, nil
}

func (a *IONOSClient) GetServer(ctx context.Context, providerID string) (*cloudprovider.InstanceMetadata, error) {
	if a.client == nil {
		return nil, errors.New("client isn't initialized")
	}
	serverReq := a.client.ServersApi.DatacentersServersFindById(ctx, a.DatacenterId, providerID)
	server, req, err := serverReq.Depth(2).Execute()
	if err != nil || req != nil && req.StatusCode == 404 {
		if err != nil {
			return nil, nil
		}
		return nil, err
	}
	return a.convertServerToInstanceMetadata(ctx, &server)
}

func (a *IONOSClient) datacenterLocation(ctx context.Context) (string, error) {
	if a.client == nil {
		return "", errors.New("client isn't initialized")
	}
	if a.cacheLocation != "" {
		return a.cacheLocation, nil
	}
	datacenter, req, err := a.client.DataCentersApi.DatacentersFindById(ctx, a.DatacenterId).Depth(2).Execute()
	if err != nil || req != nil && req.StatusCode == 404 {
		return "", err
	}
	a.cacheLocation = *datacenter.Properties.Location
	return *datacenter.Properties.Location, nil
}

func (a *IONOSClient) convertServerToInstanceMetadata(ctx context.Context, server *ionoscloud.Server) (*cloudprovider.InstanceMetadata, error) {
	if a.client == nil {
		return nil, errors.New("client isn't initialized")
	}
	location, err := a.datacenterLocation(ctx)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, nil
	}

	var addresses []v1.NodeAddress
	klog.Infof("Found %v nics", len(*server.Entities.Nics.Items))
	for _, nic := range *server.Entities.Nics.Items {
		ips := *nic.Properties.Ips
		klog.Infof("Found %v ips for nic %s. Only using the first one as the remaining ones are failover ips", len(ips), *nic.Properties.Name)
		if len(ips) > 0 {
			ipStr := ips[0]
			ip := net.ParseIP(ipStr)
			if ip == nil {
				klog.Error("Parsing failed")
				continue
			}
			var t v1.NodeAddressType
			if ip.IsPrivate() {
				t = v1.NodeInternalIP
			} else {
				t = v1.NodeExternalIP
			}
			addresses = append(addresses, v1.NodeAddress{
				Type:    t,
				Address: ipStr,
			})
		}
	}
	metadata := &cloudprovider.InstanceMetadata{
		ProviderID:    fmt.Sprintf("%s%s", config.ProviderPrefix, *server.Id),
		InstanceType:  fmt.Sprintf("dedicated-core-server.cpu-%s-%d.mem-%dmb", *server.Properties.CpuFamily, *server.Properties.Cores, *server.Properties.Ram),
		NodeAddresses: addresses,
		Zone:          *server.Properties.AvailabilityZone,
		Region:        strings.Replace(location, "/", "-", 1),
	}

	klog.InfoDepth(1, metadata)
	return metadata, err
}

func (a *IONOSClient) GetServerByName(ctx context.Context, name string) (*cloudprovider.InstanceMetadata, error) {
	klog.Infof("GetServerByName %s", name)
	if a.client == nil {
		return nil, errors.New("client is initialized")
	}
	serverReq := a.client.ServersApi.DatacentersServersGet(ctx, a.DatacenterId)
	servers, req, err := serverReq.Depth(2).Execute()
	if err != nil || servers.Items == nil || req != nil && req.StatusCode == 404 {
		if err != nil {
			return nil, nil
		}
		return nil, err
	}
	items := *servers.Items
	for i := range items {
		server := &items[i]
		if server.Properties.Name != nil && *server.Properties.Name == name {
			return a.convertServerToInstanceMetadata(ctx, server)
		}
	}
	return nil, err
}
