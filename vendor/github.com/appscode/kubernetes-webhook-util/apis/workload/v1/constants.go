package v1

import (
	"fmt"
	"strings"
)

type WorkloadKind string

const (
	KindPod                   WorkloadKind = "Pod"
	KindDeployment            WorkloadKind = "Deployment"
	KindReplicaSet            WorkloadKind = "ReplicaSet"
	KindReplicationController WorkloadKind = "ReplicationController"
	KindStatefulSet           WorkloadKind = "StatefulSet"
	KindDaemonSet             WorkloadKind = "DaemonSet"
	KindJob                   WorkloadKind = "Job"
	KindCronJob               WorkloadKind = "CronJob"
	KindDeploymentConfig      WorkloadKind = "DeploymentConfig"
)

func Canonicalize(kind string) (WorkloadKind, error) {
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
