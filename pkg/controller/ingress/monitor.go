package ingress

import (
	"fmt"

	//tapi "github.com/k8sdb/apimachinery/api"
	//"github.com/k8sdb/apimachinery/pkg/docker"
	//"github.com/k8sdb/apimachinery/pkg/monitor"
	"github.com/appscode/voyager/pkg/monitor"
)

const (
	ImageExporter = "appscode/haproxy-exporter"
)

func (c *EngressController) newMonitorController() (monitor.Monitor, error) {
	if c.Monitor == nil {
		return nil, fmt.Errorf("MonitorSpec not found in %v", c.Monitor)
	}

	if c.Monitor.Prometheus != nil {
		image := fmt.Sprintf("%v:%v", ImageExporter, c.ExporterTag)
		return monitor.NewPrometheusController(c.KubeClient, c.promClient, c.ExporterNamespace, image), nil
	}

	return nil, fmt.Errorf("Monitoring controller not found for %v", c.Monitor)
}

func (c *EngressController) addMonitor() error {
	ctrl, err := c.newMonitorController()
	if err != nil {
		return err
	}
	return ctrl.AddMonitor(postgres.ObjectMeta, c.Monitor)
}

func (c *EngressController) deleteMonitor() error {
	ctrl, err := c.newMonitorController()
	if err != nil {
		return err
	}
	return ctrl.DeleteMonitor(postgres.ObjectMeta, postgres.Spec.Monitor)
}

func (c *EngressController) updateMonitor(oldPostgres, updatedPostgres *tapi.Postgres) error {
	var err error
	var ctrl monitor.Monitor
	if updatedPostgres.Spec.Monitor == nil {
		ctrl, err = c.newMonitorController(oldPostgres)
	} else {
		ctrl, err = c.newMonitorController(updatedPostgres)
	}
	if err != nil {
		return err
	}
	return ctrl.UpdateMonitor(updatedPostgres.ObjectMeta, oldPostgres.Spec.Monitor, updatedPostgres.Spec.Monitor)
}
