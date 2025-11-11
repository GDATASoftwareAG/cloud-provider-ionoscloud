package ionos

import (
	"math/rand"

	"github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/client"
	"github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/config"
)

type IONOS struct {
	config       config.Config
	instances    instances
	loadbalancer loadbalancer
}

type instances struct {
	ionosClients map[string]*client.IONOSClient
}

type loadbalancer struct {
	r            *rand.Rand
	ionosClients map[string]*client.IONOSClient
}
