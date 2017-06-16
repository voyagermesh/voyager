package stash

import (
"k8s.io/client-go/tools/cache"
)

type Storage struct {
	PodStore         cache.StoreToPodLister
	RcStore          cache.StoreToReplicationControllerLister
	ReplicaSetStore  cache.StoreToReplicaSetLister
	StatefulSetStore cache.StoreToStatefulSetLister
	DaemonSetStore   cache.StoreToDaemonSetLister
	ServiceStore     cache.StoreToServiceLister
	EndpointStore    cache.StoreToEndpointsLister
	DeploymentStore  cache.StoreToDeploymentLister
}
