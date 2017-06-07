package main

import (
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"strconv"

	hpe "github.com/appscode/haproxy_exporter/exporter"
	"github.com/appscode/pat"
	"github.com/appscode/voyager/pkg/controller/ingress"
	"github.com/orcaman/concurrent-map"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	kapi "k8s.io/kubernetes/pkg/api"
	kerr "k8s.io/kubernetes/pkg/api/errors"
)

const (
	ParamAPIGroup  = ":apiGroup"
	ParamNamespace = ":namespace"
	ParamName      = ":name"
	ParamPodIP     = ":ip"
)

var (
	selectedServerMetrics map[int]*prometheus.GaugeVec

	registerers = cmap.New() // URL.path => *prometheus.Registry
)

func DeleteRegistry(w http.ResponseWriter, r *http.Request) {
	registerers.Remove(r.URL.Path)
	w.WriteHeader(http.StatusOK)
}

func ExportMetrics(w http.ResponseWriter, r *http.Request) {
	params, found := pat.FromContext(r.Context())
	if !found {
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}
	apiGroup := params.Get(ParamAPIGroup)
	if apiGroup == "" {
		http.Error(w, "Missing parameter:"+ParamAPIGroup, http.StatusBadRequest)
		return
	}
	namespace := params.Get(ParamNamespace)
	if namespace == "" {
		http.Error(w, "Missing parameter:"+ParamNamespace, http.StatusBadRequest)
		return
	}
	name := params.Get(ParamName)
	if name == "" {
		http.Error(w, "Missing parameter:"+ParamName, http.StatusBadRequest)
		return
	}
	podIP := params.Get(ParamPodIP)
	if podIP == "" {
		http.Error(w, "Missing parameter:"+ParamPodIP, http.StatusBadRequest)
		return
	}

	switch apiGroup {
	case "extensions":
		var reg *prometheus.Registry
		if val, ok := registerers.Get(r.URL.Path); ok {
			reg = val.(*prometheus.Registry)
		} else {
			reg = prometheus.NewRegistry()
			if absent := registerers.SetIfAbsent(r.URL.Path, reg); !absent {
				r2, _ := registerers.Get(r.URL.Path)
				reg = r2.(*prometheus.Registry)
			} else {
				log.Infof("Configuring exporter for standard ingress %s in namespace %s", name, namespace)
				ingress, err := kubeClient.Extensions().Ingresses(namespace).Get(name)
				if kerr.IsNotFound(err) {
					http.NotFound(w, r)
					return
				} else if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				scrapeURL, err := getScrapeURL(ingress.ObjectMeta, podIP)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				exporter, err := hpe.NewExporter(scrapeURL, selectedServerMetrics, haProxyTimeout)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				reg.MustRegister(exporter)
				reg.MustRegister(version.NewCollector("haproxy_exporter"))
			}
		}
		promhttp.HandlerFor(reg, promhttp.HandlerOpts{}).ServeHTTP(w, r)
		return
	case "appscode.com":
		var reg *prometheus.Registry
		if val, ok := registerers.Get(r.URL.Path); ok {
			reg = val.(*prometheus.Registry)
		} else {
			reg = prometheus.NewRegistry()
			if absent := registerers.SetIfAbsent(r.URL.Path, reg); !absent {
				r2, _ := registerers.Get(r.URL.Path)
				reg = r2.(*prometheus.Registry)
			} else {
				log.Infof("Configuring exporter for appscode ingress %s in namespace %s", name, namespace)
				ingress, err := extClient.Ingress(namespace).Get(name)
				if kerr.IsNotFound(err) {
					http.NotFound(w, r)
					return
				} else if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				scrapeURL, err := getScrapeURL(ingress.ObjectMeta, podIP)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				exporter, err := hpe.NewExporter(scrapeURL, selectedServerMetrics, haProxyTimeout)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				reg.MustRegister(exporter)
				reg.MustRegister(version.NewCollector("haproxy_exporter"))
			}
		}
		promhttp.HandlerFor(reg, promhttp.HandlerOpts{}).ServeHTTP(w, r)
		return
	}
	http.NotFound(w, r)
}

func getScrapeURL(meta kapi.ObjectMeta, podIP string) (string, error) {
	if _, ok := meta.Annotations[ingress.StatsOn]; !ok {
		return "", errors.New("Stats not exposed")
	}
	statsPort := ingress.DefaultStatsPort
	if v, ok := meta.Annotations[ingress.StatsPort]; ok {
		if port, err := strconv.Atoi(v); err == nil {
			statsPort = port
		}
	}
	secretName, ok := meta.Annotations[ingress.StatsSecret]
	if !ok {
		return fmt.Sprintf("http://%s:%d?stats;csv", podIP, statsPort), nil
	}
	secret, err := kubeClient.Core().Secrets(meta.Namespace).Get(secretName)
	if err != nil {
		return "", err
	}
	userName := string(secret.Data["username"])
	passWord := string(secret.Data["password"])
	return fmt.Sprintf("http://%s:%s@%s:%d?stats;csv", userName, passWord, podIP, statsPort), nil
}
