package monitor

import (
	"github.com/appscode/voyager/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Monitor interface {
	AddMonitor(meta metav1.ObjectMeta, spec *api.MonitorSpec) error
	UpdateMonitor(meta metav1.ObjectMeta, old, new *api.MonitorSpec) error
	DeleteMonitor(meta metav1.ObjectMeta, spec *api.MonitorSpec) error
}
