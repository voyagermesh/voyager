package collector

import (
	"crypto/tls"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

const (
	Namespace = "haproxy" // For Prometheus metrics.

	// HAProxy 1.4
	// # pxname,svname,qcur,qmax,scur,smax,slim,stot,bin,bout,dreq,dresp,ereq,econ,eresp,wretr,wredis,status,weight,act,bck,chkfail,chkdown,lastchg,downtime,qlimit,pid,iid,sid,throttle,lbtot,tracked,type,rate,rate_lim,rate_max,check_status,check_code,check_duration,hrsp_1xx,hrsp_2xx,hrsp_3xx,hrsp_4xx,hrsp_5xx,hrsp_other,hanafail,req_rate,req_rate_max,req_tot,cli_abrt,srv_abrt,
	// HAProxy 1.5
	// pxname,svname,qcur,qmax,scur,smax,slim,stot,bin,bout,dreq,dresp,ereq,econ,eresp,wretr,wredis,status,weight,act,bck,chkfail,chkdown,lastchg,downtime,qlimit,pid,iid,sid,throttle,lbtot,tracked,type,rate,rate_lim,rate_max,check_status,check_code,check_duration,hrsp_1xx,hrsp_2xx,hrsp_3xx,hrsp_4xx,hrsp_5xx,hrsp_other,hanafail,req_rate,req_rate_max,req_tot,cli_abrt,srv_abrt,comp_in,comp_out,comp_byp,comp_rsp,lastsess,
	// HAProxy 1.5.19
	// pxname,svname,qcur,qmax,scur,smax,slim,stot,bin,bout,dreq,dresp,ereq,econ,eresp,wretr,wredis,status,weight,act,bck,chkfail,chkdown,lastchg,downtime,qlimit,pid,iid,sid,throttle,lbtot,tracked,type,rate,rate_lim,rate_max,check_status,check_code,check_duration,hrsp_1xx,hrsp_2xx,hrsp_3xx,hrsp_4xx,hrsp_5xx,hrsp_other,hanafail,req_rate,req_rate_max,req_tot,cli_abrt,srv_abrt,comp_in,comp_out,comp_byp,comp_rsp,lastsess,last_chk,last_agt,qtime,ctime,rtime,ttime,
	// HAProxy 1.7
	// pxname,svname,qcur,qmax,scur,smax,slim,stot,bin,bout,dreq,dresp,ereq,econ,eresp,wretr,wredis,status,weight,act,bck,chkfail,chkdown,lastchg,downtime,qlimit,pid,iid,sid,throttle,lbtot,tracked,type,rate,rate_lim,rate_max,check_status,check_code,check_duration,hrsp_1xx,hrsp_2xx,hrsp_3xx,hrsp_4xx,hrsp_5xx,hrsp_other,hanafail,req_rate,req_rate_max,req_tot,cli_abrt,srv_abrt,comp_in,comp_out,comp_byp,comp_rsp,lastsess,last_chk,last_agt,qtime,ctime,rtime,ttime,agent_status,agent_code,agent_duration,check_desc,agent_desc,check_rise,check_fall,check_health,agent_rise,agent_fall,agent_health,addr,cookie,mode,algo,conn_rate,conn_rate_max,conn_tot,intercepted,dcon,dses
	minimumCsvFieldCount = 33
	statusField          = 17
	qtimeMsField         = 58
	ctimeMsField         = 59
	rtimeMsField         = 60
	ttimeMsField         = 61
)

var (
	frontendLabelNames = []string{"frontend"}
	backendLabelNames  = []string{"backend"}
	serverLabelNames   = []string{"backend", "server"}
)

func newServerGaugeOpts(metricName string, docString string, constLabels prometheus.Labels) prometheus.GaugeOpts {
	return prometheus.GaugeOpts{
		Namespace:   Namespace,
		Name:        "server_" + metricName,
		Help:        docString,
		ConstLabels: constLabels,
	}
}

type Metrics map[int]prometheus.GaugeOpts

func (m Metrics) String() string {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	s := make([]string, len(keys))
	for i, k := range keys {
		s[i] = strconv.Itoa(k)
	}
	return strings.Join(s, ",")
}

var (
	ServerMetrics = Metrics{
		2:  newServerGaugeOpts("current_queue", "Current number of queued requests assigned to this server.", nil),
		3:  newServerGaugeOpts("max_queue", "Maximum observed number of queued requests assigned to this server.", nil),
		4:  newServerGaugeOpts("current_sessions", "Current number of active sessions.", nil),
		5:  newServerGaugeOpts("max_sessions", "Maximum observed number of active sessions.", nil),
		6:  newServerGaugeOpts("limit_sessions", "Configured session limit.", nil),
		7:  newServerGaugeOpts("sessions_total", "Total number of sessions.", nil),
		8:  newServerGaugeOpts("bytes_in_total", "Current total of incoming bytes.", nil),
		9:  newServerGaugeOpts("bytes_out_total", "Current total of outgoing bytes.", nil),
		13: newServerGaugeOpts("connection_errors_total", "Total of connection errors.", nil),
		14: newServerGaugeOpts("response_errors_total", "Total of response errors.", nil),
		15: newServerGaugeOpts("retry_warnings_total", "Total of retry warnings.", nil),
		16: newServerGaugeOpts("redispatch_warnings_total", "Total of redispatch warnings.", nil),
		17: newServerGaugeOpts("up", "Current health status of the server (1 = UP, 0 = DOWN).", nil),
		18: newServerGaugeOpts("weight", "Current weight of the server.", nil),
		21: newServerGaugeOpts("check_failures_total", "Total number of failed health checks.", nil),
		24: newServerGaugeOpts("downtime_seconds_total", "Total downtime in seconds.", nil),
		33: newServerGaugeOpts("current_session_rate", "Current number of sessions per second over last elapsed second.", nil),
		35: newServerGaugeOpts("max_session_rate", "Maximum observed number of sessions per second.", nil),
		38: newServerGaugeOpts("check_duration_milliseconds", "Previously run health check duration, in milliseconds", nil),
		39: newServerGaugeOpts("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "1xx"}),
		40: newServerGaugeOpts("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "2xx"}),
		41: newServerGaugeOpts("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "3xx"}),
		42: newServerGaugeOpts("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "4xx"}),
		43: newServerGaugeOpts("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "5xx"}),
		44: newServerGaugeOpts("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "other"}),
	}
)

// Exporter collects HAProxy stats from the given URI and exports them using
// the prometheus metrics package.
type Exporter struct {
	URI   string
	mutex sync.RWMutex
	fetch func() (io.ReadCloser, error)

	extraLabels                                    prometheus.Labels
	up                                             prometheus.Gauge
	totalScrapes, csvParseFailures                 prometheus.Counter
	frontendMetrics, backendMetrics, serverMetrics map[int]*prometheus.GaugeVec
}

// NewExporter returns an initialized Exporter.
func NewExporter(uri string, sslVerify bool, serverMetricFields string, extraLabels prometheus.Labels, timeout time.Duration) (*Exporter, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	var fetch func() (io.ReadCloser, error)
	switch u.Scheme {
	case "http", "https", "file":
		fetch = fetchHTTP(uri, sslVerify, timeout)
	case "unix":
		fetch = fetchUnix(u, timeout)
	default:
		return nil, fmt.Errorf("unsupported scheme: %q", u.Scheme)
	}

	e := &Exporter{
		URI:         uri,
		extraLabels: extraLabels,
		fetch:       fetch,
	}
	e.up = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   Namespace,
		Name:        "up",
		Help:        "Was the last scrape of haproxy successful.",
		ConstLabels: nil,
	})
	e.totalScrapes = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   Namespace,
		Name:        "exporter_total_scrapes",
		Help:        "Current total HAProxy scrapes.",
		ConstLabels: nil,
	})
	e.csvParseFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   Namespace,
		Name:        "exporter_csv_parse_failures",
		Help:        "Number of errors while parsing CSV.",
		ConstLabels: nil,
	})
	e.frontendMetrics = map[int]*prometheus.GaugeVec{
		4:  e.newFrontendMetric("current_sessions", "Current number of active sessions.", nil),
		5:  e.newFrontendMetric("max_sessions", "Maximum observed number of active sessions.", nil),
		6:  e.newFrontendMetric("limit_sessions", "Configured session limit.", nil),
		7:  e.newFrontendMetric("sessions_total", "Total number of sessions.", nil),
		8:  e.newFrontendMetric("bytes_in_total", "Current total of incoming bytes.", nil),
		9:  e.newFrontendMetric("bytes_out_total", "Current total of outgoing bytes.", nil),
		10: e.newFrontendMetric("requests_denied_total", "Total of requests denied for security.", nil),
		12: e.newFrontendMetric("request_errors_total", "Total of request errors.", nil),
		33: e.newFrontendMetric("current_session_rate", "Current number of sessions per second over last elapsed second.", nil),
		34: e.newFrontendMetric("limit_session_rate", "Configured limit on new sessions per second.", nil),
		35: e.newFrontendMetric("max_session_rate", "Maximum observed number of sessions per second.", nil),
		39: e.newFrontendMetric("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "1xx"}),
		40: e.newFrontendMetric("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "2xx"}),
		41: e.newFrontendMetric("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "3xx"}),
		42: e.newFrontendMetric("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "4xx"}),
		43: e.newFrontendMetric("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "5xx"}),
		44: e.newFrontendMetric("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "other"}),
		48: e.newFrontendMetric("http_requests_total", "Total HTTP requests.", nil),
		79: e.newFrontendMetric("connections_total", "Total number of connections", nil),
	}
	e.backendMetrics = map[int]*prometheus.GaugeVec{
		2:  e.newBackendMetric("current_queue", "Current number of queued requests not assigned to any server.", nil),
		3:  e.newBackendMetric("max_queue", "Maximum observed number of queued requests not assigned to any server.", nil),
		4:  e.newBackendMetric("current_sessions", "Current number of active sessions.", nil),
		5:  e.newBackendMetric("max_sessions", "Maximum observed number of active sessions.", nil),
		6:  e.newBackendMetric("limit_sessions", "Configured session limit.", nil),
		7:  e.newBackendMetric("sessions_total", "Total number of sessions.", nil),
		8:  e.newBackendMetric("bytes_in_total", "Current total of incoming bytes.", nil),
		9:  e.newBackendMetric("bytes_out_total", "Current total of outgoing bytes.", nil),
		13: e.newBackendMetric("connection_errors_total", "Total of connection errors.", nil),
		14: e.newBackendMetric("response_errors_total", "Total of response errors.", nil),
		15: e.newBackendMetric("retry_warnings_total", "Total of retry warnings.", nil),
		16: e.newBackendMetric("redispatch_warnings_total", "Total of redispatch warnings.", nil),
		17: e.newBackendMetric("up", "Current health status of the backend (1 = UP, 0 = DOWN).", nil),
		18: e.newBackendMetric("weight", "Total weight of the servers in the backend.", nil),
		19: e.newBackendMetric("current_server", "Current number of active servers", nil),
		33: e.newBackendMetric("current_session_rate", "Current number of sessions per second over last elapsed second.", nil),
		35: e.newBackendMetric("max_session_rate", "Maximum number of sessions per second.", nil),
		39: e.newBackendMetric("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "1xx"}),
		40: e.newBackendMetric("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "2xx"}),
		41: e.newBackendMetric("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "3xx"}),
		42: e.newBackendMetric("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "4xx"}),
		43: e.newBackendMetric("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "5xx"}),
		44: e.newBackendMetric("http_responses_total", "Total of HTTP responses.", prometheus.Labels{"code": "other"}),
		58: e.newBackendMetric("http_queue_time_average_seconds", "Avg. HTTP queue time for last 1024 successful connections.", nil),
		59: e.newBackendMetric("http_connect_time_average_seconds", "Avg. HTTP connect time for last 1024 successful connections.", nil),
		60: e.newBackendMetric("http_response_time_average_seconds", "Avg. HTTP response time for last 1024 successful connections.", nil),
		61: e.newBackendMetric("http_total_time_average_seconds", "Avg. HTTP total time for last 1024 successful connections.", nil),
	}

	selected, err := e.filterServerMetrics(serverMetricFields)
	if err != nil {
		return nil, err
	}
	e.serverMetrics = map[int]*prometheus.GaugeVec{}
	for field, opts := range ServerMetrics {
		if _, ok := selected[field]; ok {
			e.serverMetrics[field] = e.newServerMetric(opts)
		}
	}

	return e, err
}

func (e *Exporter) newFrontendMetric(metricName string, docString string, constLabels prometheus.Labels) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   Namespace,
			Name:        "frontend_" + metricName,
			Help:        docString,
			ConstLabels: e.getConstLabels(constLabels),
		},
		frontendLabelNames,
	)
}

func (e *Exporter) newBackendMetric(metricName string, docString string, constLabels prometheus.Labels) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   Namespace,
			Name:        "backend_" + metricName,
			Help:        docString,
			ConstLabels: e.getConstLabels(constLabels),
		},
		backendLabelNames,
	)
}

func (e *Exporter) newServerMetric(opts prometheus.GaugeOpts) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   opts.Namespace,
			Name:        opts.Name,
			Help:        opts.Help,
			ConstLabels: e.getConstLabels(opts.ConstLabels),
		},
		serverLabelNames,
	)
}

func (e *Exporter) getConstLabels(constLabels prometheus.Labels) prometheus.Labels {
	result := prometheus.Labels{}
	for k, v := range constLabels {
		result[k] = v
	}
	if e != nil && e.extraLabels != nil {
		for k, v := range e.extraLabels {
			result[k] = v
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// Describe describes all the metrics ever exported by the HAProxy exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range e.frontendMetrics {
		m.Describe(ch)
	}
	for _, m := range e.backendMetrics {
		m.Describe(ch)
	}
	for _, m := range e.serverMetrics {
		m.Describe(ch)
	}
	ch <- e.up.Desc()
	ch <- e.totalScrapes.Desc()
	ch <- e.csvParseFailures.Desc()
}

// Collect fetches the stats from configured HAProxy location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()

	e.resetMetrics()
	e.scrape()

	ch <- e.up
	ch <- e.totalScrapes
	ch <- e.csvParseFailures
	e.collectMetrics(ch)
}

func fetchHTTP(uri string, sslVerify bool, timeout time.Duration) func() (io.ReadCloser, error) {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: !sslVerify}}
	client := http.Client{
		Timeout:   timeout,
		Transport: tr,
	}

	return func() (io.ReadCloser, error) {
		resp, err := client.Get(uri)
		if err != nil {
			return nil, err
		}
		if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
			resp.Body.Close()
			return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
		}
		return resp.Body, nil
	}
}

func fetchUnix(u *url.URL, timeout time.Duration) func() (io.ReadCloser, error) {
	return func() (io.ReadCloser, error) {
		f, err := net.DialTimeout("unix", u.Path, timeout)
		if err != nil {
			return nil, err
		}
		if err := f.SetDeadline(time.Now().Add(timeout)); err != nil {
			f.Close()
			return nil, err
		}
		cmd := "show stat\n"
		n, err := io.WriteString(f, cmd)
		if err != nil {
			f.Close()
			return nil, err
		}
		if n != len(cmd) {
			f.Close()
			return nil, errors.New("write error")
		}
		return f, nil
	}
}

func (e *Exporter) scrape() {
	e.totalScrapes.Inc()

	body, err := e.fetch()
	if err != nil {
		e.up.Set(0)
		log.Errorf("Can't scrape HAProxy: %v", err)
		return
	}
	defer body.Close()
	e.up.Set(1)

	reader := csv.NewReader(body)
	reader.TrailingComma = true
	reader.Comment = '#'

loop:
	for {
		row, err := reader.Read()
		switch err {
		case nil:
		case io.EOF:
			break loop
		default:
			if _, ok := err.(*csv.ParseError); ok {
				log.Errorf("Can't read CSV: %v", err)
				e.csvParseFailures.Inc()
				continue loop
			}
			log.Errorf("Unexpected error while reading CSV: %v", err)
			e.up.Set(0)
			break loop
		}
		e.parseRow(row)
	}
}

func (e *Exporter) resetMetrics() {
	for _, m := range e.frontendMetrics {
		m.Reset()
	}
	for _, m := range e.backendMetrics {
		m.Reset()
	}
	for _, m := range e.serverMetrics {
		m.Reset()
	}
}

func (e *Exporter) collectMetrics(metrics chan<- prometheus.Metric) {
	for _, m := range e.frontendMetrics {
		m.Collect(metrics)
	}
	for _, m := range e.backendMetrics {
		m.Collect(metrics)
	}
	for _, m := range e.serverMetrics {
		m.Collect(metrics)
	}
}

func (e *Exporter) parseRow(csvRow []string) {
	if len(csvRow) < minimumCsvFieldCount {
		log.Errorf("Parser expected at least %d CSV fields, but got: %d", minimumCsvFieldCount, len(csvRow))
		e.csvParseFailures.Inc()
		return
	}

	pxname, svname, typ := csvRow[0], csvRow[1], csvRow[32]

	const (
		frontend = "0"
		backend  = "1"
		server   = "2"
		listener = "3"
	)

	switch typ {
	case frontend:
		e.exportCsvFields(e.frontendMetrics, csvRow, pxname)
	case backend:
		e.exportCsvFields(e.backendMetrics, csvRow, pxname)
	case server:
		e.exportCsvFields(e.serverMetrics, csvRow, pxname, svname)
	}
}

func parseStatusField(value string) int64 {
	switch value {
	case "UP", "UP 1/3", "UP 2/3", "OPEN", "no check":
		return 1
	case "DOWN", "DOWN 1/2", "NOLB", "MAINT":
		return 0
	}
	return 0
}

func (e *Exporter) exportCsvFields(metrics map[int]*prometheus.GaugeVec, csvRow []string, labels ...string) {
	for fieldIdx, metric := range metrics {
		if fieldIdx > len(csvRow)-1 {
			break
		}
		valueStr := csvRow[fieldIdx]
		if valueStr == "" {
			continue
		}

		var err error = nil
		var value float64
		var valueInt int64

		switch fieldIdx {
		case statusField:
			valueInt = parseStatusField(valueStr)
			value = float64(valueInt)
		case qtimeMsField, ctimeMsField, rtimeMsField, ttimeMsField:
			value, err = strconv.ParseFloat(valueStr, 64)
			value /= 1000
		default:
			valueInt, err = strconv.ParseInt(valueStr, 10, 64)
			value = float64(valueInt)
		}
		if err != nil {
			log.Errorf("Can't parse CSV field value %s: %v", valueStr, err)
			e.csvParseFailures.Inc()
			continue
		}
		metric.WithLabelValues(labels...).Set(value)
	}
}

// FilterServerMetrics returns the set of server metrics specified by the comma
// separated filter.
func (e *Exporter) filterServerMetrics(filter string) (map[int]prometheus.GaugeOpts, error) {
	metrics := map[int]prometheus.GaugeOpts{}
	if len(filter) == 0 {
		return metrics, nil
	}

	selected := map[int]struct{}{}
	for _, f := range strings.Split(filter, ",") {
		field, err := strconv.Atoi(f)
		if err != nil {
			return nil, fmt.Errorf("invalid server metric field number: %v", f)
		}
		selected[field] = struct{}{}
	}

	for field, opts := range ServerMetrics {
		if _, ok := selected[field]; ok {
			metrics[field] = opts
		}
	}
	return metrics, nil
}
