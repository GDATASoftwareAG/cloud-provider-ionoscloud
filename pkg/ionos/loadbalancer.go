package ionos

import (
	"context"
	"errors"
	client2 "github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/client"
	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
	"math/rand"
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
	//TODO check if any node has service.spec.loadbalancerip
	// TODO check spec.loadbalancerclass + use .spec.loadbalancerip
	if service.Spec.Type != v1.ServiceTypeLoadBalancer {
		return nil, false, errors.New("NotFound")
	}

	if service.Spec.LoadBalancerIP == "" {
		return nil, false, errors.New("NotFound")
	}

	panic("implement me")
}

// GetLoadBalancerName returns the name of the load balancer. Implementations must treat the
// *v1.Service parameter as read-only and not modify it.
func (l loadbalancer) GetLoadBalancerName(ctx context.Context, clusterName string, service *v1.Service) string {
	//TODO return service.metadata.uid

	panic("implement me")
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
	//TODO check if ip already attached to some node,  attach loadbalancerIP to some node (not on controlplanes)
	server, err := l.GetServerWithLoadBalancer(ctx, service.Spec.LoadBalancerIP)
	if err != nil {
		return nil, err
	}

	if server != nil {
		node := getNode(*server, nodes)
		if IsLoadBalancerCandidate(node) {
			return &v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{v1.LoadBalancerIngress{
				IP: service.Spec.LoadBalancerIP,
			}}}, nil
		}
	}

	loadBalancerNode := l.GetLoadBalancerNode(nodes)

	if loadBalancerNode == nil {
		return nil, errors.New("No valid Nodes found")
	}

	for _, client := range l.ionosClients {
		ok, err := client.AttachIPToNode(ctx, service.Spec.LoadBalancerIP, loadBalancerNode.Spec.ProviderID)
		if err != nil {
			return nil, err
		}

		if ok {
			return &v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{v1.LoadBalancerIngress{
				IP: service.Spec.LoadBalancerIP,
			}}}, nil
		}
	}

	return nil, nil
}

func getNode(server string, nodes []*v1.Node) *v1.Node {
	for _, node := range nodes {
		if node.Name == server {
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

	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(candidates))
	return candidates[randomIndex]
}

func (l loadbalancer) GetServerWithLoadBalancer(ctx context.Context, loadBalancerIP string) (*string, error) {
	for _, client := range l.ionosClients {
		server, err := client.GetServerByIP(ctx, loadBalancerIP)
		if err != nil {
			return nil, err
		}

		if server != nil {
			return server, nil
		}
	}

	return nil, nil
}

func IsLoadBalancerCandidate(node *v1.Node) bool {
	if IsControlPlane(node) {
		return false
	}

	for _, condition := range node.Status.Conditions {
		if condition.Type == v1.NodeReady && condition.Status == v1.ConditionTrue {
			return true
		}
		return false
	}
	return false
}

func IsControlPlane(node *v1.Node) bool {
	for _, taint := range node.Spec.Taints {
		if taint.Key == "node-role.kubernetes.io/control-plane" {
			return true
		}
	}
	return false
}

// UpdateLoadBalancer updates hosts under the specified load balancer.
// Implementations must treat the *v1.Service and *v1.Node
// parameters as read-only and not modify them.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (l loadbalancer) UpdateLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) error {
	//TODO same as EnsureLoadBalancer
	panic("implement me")
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
	//TODO remove ip from node
	panic("implement me")
}
