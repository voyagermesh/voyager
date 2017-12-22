package v1beta1

import (
	"fmt"

	"github.com/appscode/kube-mon/api"
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

func (s statsService) GetNamespace() string {
	return s.ing.Namespace
}

func (s statsService) ServiceName() string {
	return s.ing.StatsServiceName()
}

func (s statsService) ServiceMonitorName() string {
	return VoyagerPrefix + s.ing.Namespace + "-" + s.ing.Name
}

func (s statsService) Path() string {
	return fmt.Sprintf("/%s/namespaces/%s/ingresses/%s/metrics", s.ing.APISchema(), s.ing.Namespace, s.ing.Name)
}

func (s statsService) Scheme() string {
	return ""
}
