package stash

import (
	apps "k8s.io/client-go/listers/apps/v1beta1"
	core "k8s.io/client-go/listers/core/v1"
	extensions "k8s.io/client-go/listers/extensions/v1beta1"
)

type Storage struct {
	PodStore         core.PodLister
	RcStore          core.ReplicationControllerLister
	ReplicaSetStore  extensions.ReplicaSetLister
	StatefulSetStore apps.StatefulSetLister
	DaemonSetStore   extensions.DaemonSetLister
	ServiceStore     core.ServiceLister
	EndpointStore    core.EndpointsLister
	DeploymentStore  extensions.DeploymentLister
}
