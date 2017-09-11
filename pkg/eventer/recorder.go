package eventer

import (
	"github.com/appscode/go/log"
	voyager "github.com/appscode/voyager/apis/voyager/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/record"
)

const (
	// Certificate Events
	EventReasonCertificateRenewFailed      = "RenewFailed"
	EventReasonCertificateRenewSuccessful  = "RenewSuccessful"
	EventReasonCertificateCreateFailed     = "CreateFailed"
	EventReasonCertificateCreateSuccessful = "CreateSuccessful"

	// Ingress Events
	EventReasonIngressHAProxyConfigCreateFailed      = "HAProxyConfigCreateFailed"
	EventReasonIngressConfigMapCreateFailed          = "ConfigMapCreateFailed"
	EventReasonIngressConfigMapCreateSuccessful      = "ConfigMapCreateSuccessful"
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
)

func NewEventRecorder(client clientset.Interface, component string) record.EventRecorder {
	// Event Broadcaster
	broadcaster := record.NewBroadcaster()
	broadcaster.StartEventWatcher(
		func(event *apiv1.Event) {
			if _, err := client.CoreV1().Events(event.Namespace).Create(event); err != nil {
				log.Errorln(err)
			}
		},
	)
	// Event Recorder
	return broadcaster.NewRecorder(api.Scheme, apiv1.EventSource{Component: component})
}

func ObjectReferenceFor(obj interface{}) runtime.Object {
	switch obj.(type) {
	case *voyager.Ingress, voyager.Ingress:
		ing := obj.(*voyager.Ingress)
		if ing.APISchema() == voyager.APISchemaIngress {
			return &apiv1.ObjectReference{
				APIVersion:      voyager.APISchemaIngress,
				Kind:            voyager.ResourceKindIngress,
				Namespace:       ing.Namespace,
				Name:            ing.Name,
				UID:             ing.UID,
				ResourceVersion: ing.ResourceVersion,
			}
		}
		return &apiv1.ObjectReference{
			APIVersion:      voyager.SchemeGroupVersion.String(),
			Kind:            voyager.ResourceKindIngress,
			Namespace:       ing.Namespace,
			Name:            ing.Name,
			UID:             ing.UID,
			ResourceVersion: ing.ResourceVersion,
		}
	case *voyager.Certificate, voyager.Certificate:
		cert := obj.(*voyager.Certificate)
		return &apiv1.ObjectReference{
			APIVersion:      voyager.SchemeGroupVersion.String(),
			Kind:            voyager.ResourceKindCertificate,
			Namespace:       cert.Namespace,
			Name:            cert.Name,
			UID:             cert.UID,
			ResourceVersion: cert.ResourceVersion,
		}
	}
	return &apiv1.ObjectReference{}
}
