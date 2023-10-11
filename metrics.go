// Simple HTTP benchmark tool
//
// @authors Minigun Maintainers
// @copyright 2020 Wayfair, LLC -- All rights reserved.

package main

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	pcm "github.com/prometheus/client_model/go"
)

// This is the main metrics config struct
type appMetrics struct {
	labels      map[string]string
	labelNames  []string
	labelValues []string

	// Counters
	channelFullEvents    *prometheus.CounterVec
	requestsSendCount    *prometheus.CounterVec
	requestsSendBytesSum *prometheus.CounterVec
	requestsSendSuccess  *prometheus.CounterVec
	requestsSendErrors   *prometheus.CounterVec
	responseBytesCount   *prometheus.CounterVec
	responseBytesSum     *prometheus.CounterVec

	// Gauges
	configWorkers       *prometheus.GaugeVec
	channelLength       *prometheus.GaugeVec
	channelConfigLength *prometheus.GaugeVec

	// Histograms
	histRequestsDuration         *prometheus.HistogramVec
	histDNSDuration              *prometheus.HistogramVec
	histConnectDuration          *prometheus.HistogramVec
	histGotFirstByteDuration     *prometheus.HistogramVec
	histTLSHandshakeDuration     *prometheus.HistogramVec
	histWroteRequestBodyDuration *prometheus.HistogramVec
	histResponseDuration         *prometheus.HistogramVec

	// Summaries
	summaryRequestsDuration         *prometheus.SummaryVec
	summaryDNSDuration              *prometheus.SummaryVec
	summaryConnectDuration          *prometheus.SummaryVec
	summaryGotFirstByteDuration     *prometheus.SummaryVec
	summaryTLSHandshakeDuration     *prometheus.SummaryVec
	summaryWroteRequestBodyDuration *prometheus.SummaryVec
	summaryResponseDuration         *prometheus.SummaryVec
}

func initMetrics(config appConfig, labelNames, labelValues []string) appMetrics {

	am := appMetrics{}
	am.labelNames = labelNames
	am.labelValues = labelValues

	am.labels = make(map[string]string, 0)
	for i := range labelNames {
		am.labels[labelNames[i]] = labelValues[i]
	}

	// Requests metrics
	am.requestsSendCount = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "minigun",
			Subsystem: "requests",
			Name:      "total",
			Help:      "The total number of requests sent to remote endpoint",
		},
		am.labelNames,
	)

	am.requestsSendBytesSum = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "minigun",
			Subsystem: "requests",
			Name:      "bytes_sum",
			Help:      "The total number of bytes received from remote endpoint",
		},
		am.labelNames,
	)

	am.requestsSendSuccess = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "minigun",
			Subsystem: "requests",
			Name:      "success_total",
			Help:      "The total number of requests successfully sent to remote endpoint",
		},
		am.labelNames,
	)

	am.requestsSendErrors = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "minigun",
			Subsystem: "requests",
			Name:      "errors_total",
			Help:      "The total number of errors when sending requests",
		},
		am.labelNames,
	)

	am.histRequestsDuration = promauto.With(registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "minigun",
			Subsystem: "requests",
			Name:      "hist_duration_seconds",
			Help:      "Histogram distribution of request durations, in seconds",
			Buckets:   secondsDurationBuckets,
		},
		am.labelNames,
	)

	am.summaryRequestsDuration = promauto.With(registry).NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "minigun",
			Subsystem:  "requests",
			Name:       "duration_seconds",
			Help:       "Summary distribution of request durations, in seconds",
			Objectives: summaryObjectives,
		},
		am.labelNames,
	)

	// DNS metrics
	am.histDNSDuration = promauto.With(registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "minigun",
			Subsystem: "httptrace",
			Name:      "hist_dns_duration_seconds",
			Help:      "Histogram distribution of DNS durations, in seconds",
			Buckets:   secondsDurationBuckets,
		},
		am.labelNames,
	)

	am.summaryDNSDuration = promauto.With(registry).NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "minigun",
			Subsystem:  "httptrace",
			Name:       "dns_duration_seconds",
			Help:       "Summary distribution of DNS durations, in seconds",
			Objectives: summaryObjectives,
		},
		am.labelNames,
	)

	// Connection metrics
	am.histConnectDuration = promauto.With(registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "minigun",
			Subsystem: "httptrace",
			Name:      "hist_connect_duration_seconds",
			Help:      "Histogram distribution of connection durations, in seconds",
			Buckets:   secondsDurationBuckets,
		},
		am.labelNames,
	)

	am.summaryConnectDuration = promauto.With(registry).NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "minigun",
			Subsystem:  "httptrace",
			Name:       "connect_duration_seconds",
			Help:       "Summary distribution of connection durations, in seconds",
			Objectives: summaryObjectives,
		},
		am.labelNames,
	)

	// Response first byte metrics
	am.histGotFirstByteDuration = promauto.With(registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "minigun",
			Subsystem: "httptrace",
			Name:      "hist_time_to_first_byte_seconds",
			Help:      "Histogram distribution of time to first byte durations, in seconds",
			Buckets:   secondsDurationBuckets,
		},
		am.labelNames,
	)

	am.summaryGotFirstByteDuration = promauto.With(registry).NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "minigun",
			Subsystem:  "httptrace",
			Name:       "time_to_first_byte_seconds",
			Help:       "Summary distribution of time to first byte durations, in seconds",
			Objectives: summaryObjectives,
		},
		am.labelNames,
	)

	// TLS handshake metrics
	am.histTLSHandshakeDuration = promauto.With(registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "minigun",
			Subsystem: "httptrace",
			Name:      "hist_tls_handshake_duration_seconds",
			Help:      "Histogram distribution of TLS Handshake durations, in seconds",
			Buckets:   secondsDurationBuckets,
		},
		am.labelNames,
	)

	am.summaryTLSHandshakeDuration = promauto.With(registry).NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "minigun",
			Subsystem:  "httptrace",
			Name:       "tls_handshake_duration_seconds",
			Help:       "Summary distribution of TLS Handshake durations, in seconds",
			Objectives: summaryObjectives,
		},
		am.labelNames,
	)

	// Request body sent metrics
	am.histWroteRequestBodyDuration = promauto.With(registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "minigun",
			Subsystem: "httptrace",
			Name:      "hist_write_request_body_duration_seconds",
			Help:      "Histogram distribution of WroteRequestBody durations, in seconds",
			Buckets:   secondsDurationBuckets,
		},
		am.labelNames,
	)

	am.summaryWroteRequestBodyDuration = promauto.With(registry).NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "minigun",
			Subsystem:  "httptrace",
			Name:       "write_request_body_duration_seconds",
			Help:       "Summary distribution of WroteRequestBody durations, in seconds",
			Objectives: summaryObjectives,
		},
		am.labelNames,
	)

	// Response metrics
	am.responseBytesCount = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "minigun",
			Subsystem: "response",
			Name:      "bytes_count",
			Help:      "The count of responses with bytes",
		},
		am.labelNames,
	)

	am.responseBytesSum = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "minigun",
			Subsystem: "response",
			Name:      "bytes_sum",
			Help:      "The sum of response bytes",
		},
		am.labelNames,
	)

	am.histResponseDuration = promauto.With(registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "minigun",
			Subsystem: "response",
			Name:      "hist_duration_seconds",
			Help:      "Histogram distribution of response durations, in seconds",
			Buckets:   secondsDurationBuckets,
		},
		append(am.labelNames, "status"),
	)

	am.summaryResponseDuration = promauto.With(registry).NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "minigun",
			Subsystem:  "response",
			Name:       "duration_seconds",
			Help:       "Summary distribution of response durations, in seconds",
			Objectives: summaryObjectives,
		},
		append(am.labelNames, "status"),
	)

	// App health metrics
	am.configWorkers = promauto.With(registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "minigun",
			Subsystem: "config",
			Name:      "workers",
			Help:      "Number of workers",
		},
		am.labelNames,
	)

	am.channelLength = promauto.With(registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "minigun",
			Subsystem: "runtime",
			Name:      "channel_length",
			Help:      "Number of messages in the main channel",
		},
		am.labelNames,
	)

	am.channelConfigLength = promauto.With(registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "minigun",
			Subsystem: "config",
			Name:      "channel_length",
			Help:      "Max channel length",
		},
		am.labelNames,
	)

	am.channelFullEvents = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "minigun",
			Subsystem: "runtime",
			Name:      "channel_full_events",
			Help:      "Number of events when worker channel was full",
		},
		am.labelNames,
	)

	am.configWorkers.WithLabelValues(labelValues...).Set(float64(config.workers))
	am.channelConfigLength.WithLabelValues(labelValues...).Set(float64(workersCannelSize))
	am.channelLength.WithLabelValues(labelValues...).Set(float64(0))
	am.channelFullEvents.WithLabelValues(labelValues...).Add(0)

	return am
}

// Get counter value
func getCounter(cp *prometheus.CounterVec, labels ...string) (float64, error) {

	c, err := cp.GetMetricWithLabelValues(labels...)
	if err != nil {
		return float64(0), err
	}

	m := &pcm.Metric{}
	c.Write(m)

	return m.GetCounter().GetValue(), nil
}

// Does the label match?
func labelMatched(labels map[string]string, labelPairs []*pcm.LabelPair) bool {
	matched := 0

	for lKey, lValue := range labels {
		for _, lp := range labelPairs {
			if lKey == lp.GetName() {
				if lValue == lp.GetValue() {
					matched++
				}
			}
		}
	}

	// If all labels matched, return true
	if matched >= len(labels) {
		return true
	}
	return false
}

// Get label values
func getSummaryLabelValues(reg *prometheus.Registry, name string, label string) ([]string, error) {
	result := make([]string, 0)

	metrics, err := prometheus.Gatherer(reg).Gather()
	if err != nil {
		return result, err
	}

	for _, mF := range metrics {
		switch *mF.Name {
		case name:
			for _, mp := range mF.Metric {
				m := *mp
				if m.Summary != nil {
					for _, lp := range m.GetLabel() {
						if lp.GetName() == label {
							result = append(result, lp.GetValue())
						}
					}
				}
			}
		}
	}

	return result, nil
}

// Get counter value by metric name
func getCountSumFromSummary(reg *prometheus.Registry, name string, labels map[string]string) (uint64, float64, error) {

	count := uint64(0)
	sum := float64(0)

	applog.Infof("DEBUG GET COUNT AND SUM FROM SUMMARY: %s", name)

	metrics, err := prometheus.Gatherer(reg).Gather()
	if err != nil {
		return uint64(0), float64(0), err
	}

	for _, mF := range metrics {
		applog.Infof("DEBUG: checking MF name: %s", *mF.Name)

		switch *mF.Name {
		case name:
			applog.Infof("DEBUG: mF.Metrics length: %v", len(mF.Metric))
			for _, mp := range mF.Metric {
				m := *mp
				if m.Summary != nil {
					if labelMatched(labels, m.GetLabel()) {
						applog.Info("DEBUG: Requested labels matched")
						count += m.Summary.GetSampleCount()
						sum += m.Summary.GetSampleSum()
					} else {
						applog.Info("DEBUG: Labels did not match")
					}
				}
			}
		}
	}

	// Return no error only if we found something
	if count > 0 {
		return count, sum, nil
	}
	return count, sum, fmt.Errorf("Metric %s not found", name)
}

// Get summary quantile values by metric name, returns a list of values
func getQuantileValuesByName(reg *prometheus.Registry, name string) ([]float64, error) {
	result := make([]float64, 0)

	metrics, err := prometheus.Gatherer(reg).Gather()
	if err != nil {
		return result, err
	}

	for _, mF := range metrics {
		switch *mF.Name {
		case name:
			m := *mF.Metric[0]
			if m.Summary != nil {
				quantiles := m.Summary.GetQuantile()
				for _, q := range quantiles {
					result = append(result, q.GetValue())
				}
			}
		}
	}

	return result, nil
}

// Get summary quantiles by metric name, returns a map of quantiles
func getQuantilesByName(reg *prometheus.Registry, name string) (map[float64]float64, error) {
	result := make(map[float64]float64)

	metrics, err := prometheus.Gatherer(reg).Gather()
	if err != nil {
		return result, err
	}

	for _, mF := range metrics {
		switch *mF.Name {
		case name:
			m := *mF.Metric[0]
			if m.Summary != nil {
				quantiles := m.Summary.GetQuantile()
				for _, q := range quantiles {
					result[q.GetQuantile()] = q.GetValue()
				}
			}
		}
	}

	return result, nil
}

// Get all values we're interested in from the summary. Returns:
//   - uint64 - count
//   - float64 - sum
//   - float64 - mean
//   - []float64 - quantiles
//   - error
func getSummaryValues(reg *prometheus.Registry, name string, labels map[string]string) (uint64, float64, float64, map[float64]float64, error) {

	var requests uint64
	var mean, seconds float64
	var quantiles map[float64]float64
	var err error

	if requests, seconds, err = getCountSumFromSummary(reg, name, labels); err == nil && requests > 0 {
		mean = seconds / float64(requests)
		quantiles, err = getQuantilesByName(reg, name)
		if err != nil {
			applog.Errorf("Got error from getQuantilesByName: %s", err.Error())
			return requests, seconds, mean, quantiles, err
		}
	}

	return requests, seconds, mean, quantiles, nil
}

// Quick helper for human readable text report
func humanizeListOfSeconds(list []float64) []string {
	result := make([]string, 0)

	for _, v := range list {
		result = append(result, humanizeDurationSeconds(v))
	}

	return result
}

// Add prometheus counter value to report matrix
func addCounterToReport(outMatrix printMatrix, name string, cp *prometheus.CounterVec, labels ...string) printMatrix {

	if counter, err := getCounter(cp, labels...); err == nil {
		outMatrix = append(outMatrix, printRow{name, fmt.Sprintf("%v", counter)})
	} else {
		applog.Errorf("Error getting Prometheus counter: %v", err.Error())
	}

	return outMatrix
}

// Add mean and quantiles to the report matrix
func addSummaryToReport(outMatrix printMatrix, header string, reg *prometheus.Registry, name string, labels map[string]string) printMatrix {

	if requests, seconds, err := getCountSumFromSummary(reg, name, labels); err == nil && requests > 0 {
		requestTime := humanizeDurationSeconds(seconds / float64(requests))
		quantiles, err := getQuantileValuesByName(reg, name)
		if err != nil {
			applog.Errorf("Got error from getQuantilesByName: %s", err.Error())
		}
		row := printRow{header, requestTime}
		row = append(row, humanizeListOfSeconds(quantiles)...)
		outMatrix = append(outMatrix, row)
	}

	return outMatrix
}
