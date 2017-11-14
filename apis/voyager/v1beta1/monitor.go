package v1beta1

import (
	"fmt"

	"github.com/appscode/kutil/tools/monitoring/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StatsPortName             = "stats"
	ExporterPortName          = "http"
	DefaultExporterPortNumber = 56790
)

func (r Ingress) StatsAccessor() api.StatsAccessor {
	return &statsService{ing: r}
}

type statsService struct {
	ing Ingress
}

func (s statsService) Namespace() string {
	return s.ing.Namespace
}

func (s statsService) ServiceName() string {
	return s.ing.StatsServiceName()
}

func (s statsService) ServiceMonitorName() string {
	return VoyagerPrefix + s.ing.Namespace + "-" + s.ing.Name
}

func (s statsService) Selector() metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: s.ing.StatsLabels(),
	}
}

func (s statsService) Path() string {
	return fmt.Sprintf("/%s/namespaces/%s/ingresses/%s/metrics", s.ing.APISchema(), s.ing.Namespace, s.ing.Name)
}

func (s statsService) Scheme() string {
	return ""
}
