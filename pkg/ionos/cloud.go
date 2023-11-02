package ionos

import (
	"context"
	"encoding/json"
	client2 "github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/client"
	"github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/config"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
)

func init() {
	cloudprovider.RegisterCloudProvider(config.RegisteredProviderName, func(cfg io.Reader) (cloudprovider.Interface, error) {
		byConfig, err := io.ReadAll(cfg)
		if err != nil {
			klog.Errorf("ReadAll failed: %s", err)
			return nil, err
		}
		var conf config.Config
		err = json.Unmarshal(byConfig, &conf)
		if err != nil {
			return nil, err
		}

		return newProvider(conf), nil
	})
}

var _ cloudprovider.Interface = &IONOS{}

func newProvider(config config.Config) cloudprovider.Interface {
	return IONOS{
		config: config,
		instances: instances{
			clients: map[string]*client2.IONOSClient{},
		},
	}
}

func (p IONOS) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, _ <-chan struct{}) {
	ctx := context.Background()
	client, err := clientBuilder.Client(config.ClientName)
	if err != nil {
		klog.Errorf("Kubernetes Client Init Failed: %v", err)
		return
	}
	secret, err := client.CoreV1().Secrets(p.config.TokenSecretNamespace).Get(ctx, p.config.TokenSecretName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get secret %s/%s: %v", p.config.TokenSecretNamespace, p.config.TokenSecretName, err)
		return
	}
	for key, token := range secret.Data {
		klog.Infof("AddClient %s", key)
		err := p.instances.AddClient(key, token)
		if err != nil {
			klog.Errorf("Failed to create client for datacenter %s: %v", key, err)
			return
		}
	}
}

func (p IONOS) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	klog.Warning("The IONOS cloud provider does not support load balancers")
	return nil, false
}

func (p IONOS) Instances() (cloudprovider.Instances, bool) {
	klog.Warning("The IONOS cloud provider does not support instances")
	return nil, false
}

func (p IONOS) InstancesV2() (cloudprovider.InstancesV2, bool) {
	return p.instances, true
}

func (p IONOS) Zones() (cloudprovider.Zones, bool) {
	klog.Warning("The IONOS cloud provider does not support zones")
	return nil, false
}

func (p IONOS) Clusters() (cloudprovider.Clusters, bool) {
	klog.Warning("The IONOS cloud provider does not support clusters")
	return nil, false
}

func (p IONOS) Routes() (cloudprovider.Routes, bool) {
	klog.Warning("The IONOS cloud provider does not support routes")
	return nil, false
}

func (p IONOS) ProviderName() string {
	return config.RegisteredProviderName
}

func (p IONOS) HasClusterID() bool {
	return true
}
