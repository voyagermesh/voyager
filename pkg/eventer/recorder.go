package eventer

import (
	"github.com/appscode/go/log"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
)

const (
	// Certificate Events
	EventReasonCertificateIssueFailed     = "IssueFailed"
	EventReasonCertificateIssueSuccessful = "IssueSuccessful"
	EventReasonCertificateInvalid         = "CertificateInvalid"
	EventReasonCertificateMigration       = "CertificateMigration"

	// Ingress Events
	EventReasonIngressConfigMapReconcileFailed        = "ConfigMapReconcileFailed"
	EventReasonIngressConfigMapReconcileSuccessful    = "ConfigMapReconcileSuccessful"
	EventReasonIngressDeploymentReconcileFailed       = "DeploymentReconcileFailed"
	EventReasonIngressDeploymentReconcileSuccessful   = "DeploymentReconcileSuccessful"
	EventReasonIngressFirewallUpdateFailed            = "FirewallUpdateFailed"
	EventReasonIngressHAProxyConfigReconcileFailed    = "HAProxyConfigFailed"
	EventReasonIngressInvalid                         = "IngressInvalid"
	EventReasonIngressMonitorAgentReconcileFailed     = "MonitorAgentReconcileFailed"
	EventReasonIngressMonitorAgentReconcileSuccessful = "MonitorAgentReconcileSuccessful"
	EventReasonIngressRBACFailed                      = "RBACFailed"
	EventReasonIngressRBACSuccessful                  = "RBACSuccessful"
	EventReasonIngressServiceReconcileFailed          = "ServiceReconcileFailed"
	EventReasonIngressServiceReconcileSuccessful      = "ServiceReconcileSuccessful"
	EventReasonIngressStatsServiceDeleteFailed        = "StatsServiceDeleteFailed"
	EventReasonIngressStatsServiceDeleteSuccessful    = "StatsServiceDeleteSuccessful"
	EventReasonIngressStatsServiceReconcileFailed     = "StatsServiceReconcileFailed"
	EventReasonIngressStatsServiceReconcileSuccessful = "StatsServiceReconcileSuccessful"
	EventReasonIngressTLSMountFailed                  = "TLSMountFailed"
)

func NewEventRecorder(client kubernetes.Interface, component string) record.EventRecorder {
	// Event Broadcaster
	broadcaster := record.NewBroadcaster()
	broadcaster.StartEventWatcher(
		func(event *core.Event) {
			if _, err := client.CoreV1().Events(event.Namespace).Create(event); err != nil {
				log.Errorln(err)
			}
		},
	)
	// Event Recorder
	return broadcaster.NewRecorder(scheme.Scheme, core.EventSource{Component: component})
}
