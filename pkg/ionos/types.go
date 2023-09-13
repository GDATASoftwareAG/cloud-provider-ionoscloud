package ionos

import (
	"github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/client"
	"github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/config"
	cloudprovider "k8s.io/cloud-provider"
)

type IONOS struct {
	config    config.Config
	instances cloudprovider.InstancesV2
	client    *client.IONOSClient
}

type instances struct {
	datacenterId string
	client       *client.IONOSClient
}
