package ionos

import (
	"github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/client"
	"github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/config"
)

type IONOS struct {
	config       config.Config
	instances    instances
	loadbalancer loadbalancer
	client       *client.IONOSClient
}

type instances struct {
	ionosClients map[string]*client.IONOSClient
}

type loadbalancer struct {
	ionosClients map[string]*client.IONOSClient
}
