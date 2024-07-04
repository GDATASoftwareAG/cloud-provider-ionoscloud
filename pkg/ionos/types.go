package ionos

import (
	"github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/client"
	"github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/config"
)

type IONOS struct {
	config    config.Config
	instances instances
	client    *client.IONOSClient
}

type instances struct {
	clients map[string]*client.IONOSClient
}
