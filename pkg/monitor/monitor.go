package monitor

import (
	api "github.com/appscode/voyager/apis/voyager"
)

type Monitor interface {
	AddMonitor(r *api.Ingress, spec *api.MonitorSpec) error
	UpdateMonitor(r *api.Ingress, old, new *api.MonitorSpec) error
	DeleteMonitor(r *api.Ingress, spec *api.MonitorSpec) error
}
