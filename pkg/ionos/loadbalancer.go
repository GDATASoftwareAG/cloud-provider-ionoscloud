package ionos

import (
	"context"
	"errors"
	client2 "github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/client"
	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
	"math/rand"
	"strings"
	"time"
)

var _ cloudprovider.LoadBalancer = &loadbalancer{}

// see https://github.com/kubernetes/kubernetes/blob/v1.18.0/pkg/controller/service/controller.go

func (l loadbalancer) AddClient(datacenterId string, token []byte) error {
	if l.ionosClients[datacenterId] == nil {
		c, err := client2.New(datacenterId, token)
		if err != nil {
			return err
		}
		l.ionosClients[datacenterId] = &c
	}
	return nil
}

// GetLoadBalancer returns whether the specified load balancer exists, and
// if so, what its status is.
// Implementations must treat the *v1.Service parameter as read-only and not modify it.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager.
// For the given LB service, the GetLoadBalancer must return "exists=True" if
// there exists a LoadBalancer instance created by ServiceController.
// In all other cases, GetLoadBalancer must return a NotFound error.
func (l loadbalancer) GetLoadBalancer(ctx context.Context, clusterName string, service *v1.Service) (status *v1.LoadBalancerStatus, exists bool, err error) {
	klog.Infof("getLoadBalancer (service %s/%s)", service.Namespace, service.Name)

	server, err := l.ServerWithLoadBalancer(ctx, service.Spec.LoadBalancerIP)
	if err != nil {
		return nil, false, err
	}

	if server != nil {
		return &v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: service.Spec.LoadBalancerIP}}}, true, nil
	}

	return nil, false, nil
}

// GetLoadBalancerName returns the name of the load balancer. Implementations must treat the
// *v1.Service parameter as read-only and not modify it.
func (l loadbalancer) GetLoadBalancerName(ctx context.Context, clusterName string, service *v1.Service) string {
	return cloudprovider.DefaultLoadBalancerName(service)
}

// EnsureLoadBalancer creates a new load balancer 'name', or updates the existing one. Returns the status of the balancer
// Implementations must treat the *v1.Service and *v1.Node
// parameters as read-only and not modify them.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager.
//
// Implementations may return a (possibly wrapped) api.RetryError to enforce
// backing off at a fixed duration. This can be used for cases like when the
// load balancer is not ready yet (e.g., it is still being provisioned) and
// polling at a fixed rate is preferred over backing off exponentially in
// order to minimize latency.
func (l loadbalancer) EnsureLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {
	return l.syncLoadBalancer(ctx, clusterName, service, nodes)
}

// UpdateLoadBalancer updates hosts under the specified load balancer.
// Implementations must treat the *v1.Service and *v1.Node
// parameters as read-only and not modify them.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (l loadbalancer) UpdateLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) error {
	_, err := l.syncLoadBalancer(ctx, clusterName, service, nodes)
	return err
}

// EnsureLoadBalancerDeleted deletes the specified load balancer if it
// exists, returning nil if the load balancer specified either didn't exist or
// was successfully deleted.
// This construction is useful because many cloud providers' load balancers
// have multiple underlying components, meaning a Get could say that the LB
// doesn't exist even if some part of it is still laying around.
// Implementations must treat the *v1.Service parameter as read-only and not modify it.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
// EnsureLoadBalancerDeleted must not return ImplementedElsewhere to ensure
// proper teardown of resources that were allocated by the ServiceController.
func (l loadbalancer) EnsureLoadBalancerDeleted(ctx context.Context, clusterName string, service *v1.Service) error {
	klog.Infof("ensureLoadBalancerDeleted (service %s/%s)", service.Namespace, service.Name)

	if service.Spec.LoadBalancerIP == "" {
		return errors.New("we are only handling LoadBalancers with spec.loadBalancerID != ''")
	}

	server, err := l.ServerWithLoadBalancer(ctx, service.Spec.LoadBalancerIP)
	if err != nil {
		return err
	}

	if server != nil {
		return l.deleteLoadBalancerFromNode(ctx, service.Spec.LoadBalancerIP, server)
	}

	return nil
}

func (l loadbalancer) deleteLoadBalancerFromNode(ctx context.Context, loadBalancerIP string, server *client2.Server) error {
	for _, client := range l.ionosClients {
		if client.DatacenterId != server.DatacenterID {
			continue
		}

		server, err := client.GetServerByIP(ctx, loadBalancerIP)
		if err != nil {
			return err
		}

		if server != nil {
			return client.RemoveIPFromNode(ctx, loadBalancerIP, server.ProviderID)
		}
	}

	klog.Infof("IP %s not found in any datacenter", loadBalancerIP)
	return nil
}

func (l loadbalancer) syncLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {
	klog.Infof("syncLoadBalancer (service %s/%s, nodes %s)", service.Namespace, service.Name, nodes)

	if service.Spec.LoadBalancerIP == "" {
		return nil, errors.New("we are only handling LoadBalancers with spec.loadBalancerID != ''")
	}

	server, err := l.ServerWithLoadBalancer(ctx, service.Spec.LoadBalancerIP)
	if err != nil {
		return nil, err
	}

	if server != nil {
		klog.Infof("found server %s has IP %s ", server, service.Spec.LoadBalancerIP)
		node := getNode(*server, nodes)
		if node == nil {
			return nil, errors.New("no node found for server which has loadbalancerIP attached")
		}

		if IsLoadBalancerCandidate(node) {
			klog.Infof("server %s is valid loadbalancer node", server)
			return &v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{
				IP: service.Spec.LoadBalancerIP,
			}}}, nil
		}
	}

	loadBalancerNode := l.GetLoadBalancerNode(nodes)

	if loadBalancerNode == nil {
		return nil, errors.New("no valid nodes found")
	}
	klog.Infof("server %s is elected as new loadbalancer node", server)

	for _, client := range l.ionosClients {
		ok, err := client.AttachIPToNode(ctx, service.Spec.LoadBalancerIP, stripProviderFromID(loadBalancerNode.Spec.ProviderID))
		if err != nil {
			return nil, err
		}

		if ok {
			klog.Infof("successfully attached ip %s to server %s", service.Spec.LoadBalancerIP, server)
			return &v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{
				IP: service.Spec.LoadBalancerIP,
			}}}, nil
		}
	}

	klog.Infof("Could not attach ip %s to any node", service.Spec.LoadBalancerIP)
	return nil, nil
}

func getNode(server client2.Server, nodes []*v1.Node) *v1.Node {
	for _, node := range nodes {
		if stripProviderFromID(node.Spec.ProviderID) == server.ProviderID {
			return node
		}
	}
	return nil
}

func (l loadbalancer) GetLoadBalancerNode(nodes []*v1.Node) *v1.Node {
	var candidates []*v1.Node
	for _, node := range nodes {
		if IsLoadBalancerCandidate(node) {
			candidates = append(candidates, node)
		}
	}
	if candidates == nil && len(candidates) == 0 {
		return nil
	}
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(candidates))
	return candidates[randomIndex]
}

func (l loadbalancer) ServerWithLoadBalancer(ctx context.Context, loadBalancerIP string) (*client2.Server, error) {
	for _, client := range l.ionosClients {
		server, err := client.GetServerByIP(ctx, loadBalancerIP)
		if err != nil {
			return nil, err
		}

		if server != nil {
			return server, nil
		}
	}

	klog.Infof("IP %s not found in any datacenter", loadBalancerIP)
	return nil, nil
}

func IsLoadBalancerCandidate(node *v1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == v1.NodeReady && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

func stripProviderFromID(providerID string) string {
	s, _ := strings.CutPrefix(providerID, "ionos://")
	return s
}
