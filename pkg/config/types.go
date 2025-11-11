package config

const (
	// RegisteredProviderName is the name of the cloud provider registered with
	// Kubernetes.
	RegisteredProviderName = "ionos"
	ProviderPrefix         = "ionos://"
	// ClientName is the user agent passed into the controller client builder.
	ClientName = "ionoscloud-cloud-controller-manager"
)

type Config struct {
	TokenSecretName      string `json:"tokenSecretName"`
	TokenSecretNamespace string `json:"tokenSecretNamespace"`
}
