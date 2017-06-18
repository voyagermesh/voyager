package monitor

import (
	"github.com/appscode/voyager/api"
)

type Monitor interface {
	AddMonitor(r *api.Ingress, spec *api.MonitorSpec) error
	UpdateMonitor(r *api.Ingress, old, new *api.MonitorSpec) error
	DeleteMonitor(r *api.Ingress, spec *api.MonitorSpec) error
}
