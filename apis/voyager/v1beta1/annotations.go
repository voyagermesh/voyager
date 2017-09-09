package v1beta1

import (
	"strings"

	"github.com/appscode/voyager/apis/voyager"
)

func (r Ingress) APISchema() string {
	if v := voyager.GetString(r.Annotations, voyager.APISchema); v != "" {
		return v
	}
	return voyager.APISchemaEngress
}

func (r Ingress) OffshootName() string {
	return voyager.VoyagerPrefix + r.Name
}

func (r Ingress) OffshootLabels() map[string]string {
	lbl := map[string]string{
		"origin":      "voyager",
		"origin-name": r.Name,
	}

	gv := strings.SplitN(r.APISchema(), "/", 2)
	if len(gv) == 2 {
		lbl["origin-api-group"] = gv[0]
	}
	return lbl
}

func (r Ingress) LBType() string {
	if v := voyager.GetString(r.Annotations, voyager.LBType); v != "" {
		return v
	}
	return voyager.LBTypeLoadBalancer
}

func (r Ingress) NodeSelector() map[string]string {
	if v, _ := voyager.GetMap(r.Annotations, voyager.NodeSelector); len(v) > 0 {
		return v
	}
	return voyager.ParseDaemonNodeSelector(voyager.GetString(r.Annotations, voyager.EngressKey+"/"+"daemon.nodeSelector"))
}

func (r Ingress) StatsServiceName() string {
	if v := voyager.GetString(r.Annotations, voyager.StatsServiceName); v != "" {
		return v
	}
	return voyager.VoyagerPrefix + r.Name + "-stats"
}
