package v1

import (
	"fmt"
	"strings"
)

const (
	KindPod                   = "Pod"
	KindDeployment            = "Deployment"
	KindReplicaSet            = "ReplicaSet"
	KindReplicationController = "ReplicationController"
	KindStatefulSet           = "StatefulSet"
	KindDaemonSet             = "DaemonSet"
	KindJob                   = "Job"
	KindCronJob               = "CronJob"
	KindDeploymentConfig      = "DeploymentConfig"
)

func Canonicalize(kind string) (string, error) {
	switch strings.ToLower(kind) {
	case "deployments", "deployment", "deploy":
		return KindDeployment, nil
	case "replicasets", "replicaset", "rs":
		return KindReplicaSet, nil
	case "replicationcontrollers", "replicationcontroller", "rc":
		return KindReplicationController, nil
	case "statefulsets", "statefulset":
		return KindStatefulSet, nil
	case "daemonsets", "daemonset", "ds":
		return KindDaemonSet, nil
	case "jobs", "job":
		return KindJob, nil
	case "cronjobs", "cronjob":
		return KindCronJob, nil
	case "deploymentconfigs", "deploymentconfig":
		return KindDeploymentConfig, nil
	default:
		return "", fmt.Errorf(`unrecognized workload "Kind" %v`, kind)
	}
}
