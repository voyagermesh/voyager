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
	EventReasonIngressHAProxyConfigCreateFailed      = "HAProxyConfigCreateFailed"
	EventReasonIngressConfigMapCreateFailed          = "ConfigMapCreateFailed"
	EventReasonIngressConfigMapCreateSuccessful      = "ConfigMapCreateSuccessful"
	EventReasonIngressRBACFailed                     = "RBACFailed"
	EventReasonIngressRBACSuccessful                 = "RBACSuccessful"
	EventReasonIngressUnsupportedLBType              = "UnsupportedLBType"
	EventReasonIngressControllerCreateFailed         = "ControllerCreateFailed"
	EventReasonIngressControllerCreateSuccessful     = "ControllerCreateSuccessful"
	EventReasonIngressServiceCreateFailed            = "ServiceCreateFailed"
	EventReasonIngressServiceCreateSuccessful        = "ServiceCreateSuccessful"
	EventReasonIngressServiceMonitorCreateFailed     = "ServiceMonitorCreateFailed"
	EventReasonIngressServiceMonitorCreateSuccessful = "ServiceMonitorCreateSuccessful"
	EventReasonIngressUpdateFailed                   = "UpdateFailed"
	EventReasonIngressDeleteFailed                   = "DeleteFailed"
	EventReasonIngressUpdateSuccessful               = "UpdateSuccessful"
	EventReasonIngressServiceUpdateFailed            = "ServiceUpdateFailed"
	EventReasonIngressServiceUpdateSuccessful        = "ServiceUpdateSuccessful"
	EventReasonIngressFirewallUpdateFailed           = "FirewallUpdateFailed"
	EventReasonIngressStatsServiceCreateFailed       = "StatsServiceCreateFailed"
	EventReasonIngressStatsServiceCreateSuccessful   = "StatsServiceCreateSuccessful"
	EventReasonIngressStatsServiceDeleteFailed       = "StatsServiceDeleteFailed"
	EventReasonIngressStatsServiceDeleteSuccessful   = "StatsServiceDeleteSuccessful"
	EventReasonIngressInvalid                        = "IngressInvalid"
	EventReasonIngressTLSMountFailed                 = "TLSMountFailed"
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
