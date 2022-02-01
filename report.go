// Simple HTTP benchmark tool
//
// @authors Minigun Maintainers
// @copyright 2020 Wayfair, LLC -- All rights reserved.

package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/olekukonko/tablewriter"
)

// Types for report printing
type printMatrix [][]string
type printRow []string

// Main report structure
type appReport struct {
	Target          string  `json:"Target"`
	SendMethod      string  `json:"SendMethod"`
	DurationSeconds float64 `json:"DurationSeconds"`
	MaxConcurrency  int     `json:"MaxConcurrency"`
	RequestBodySize int64   `json:"RequestBodySize"`

	RequestsCompleted float64 `json:"RequestsCompleted"`
	RequestsSucceeded float64 `json:"RequestsSucceeded"`
	RequestsFailed    float64 `json:"RequestsFailed"`

	OverallRequestsRate           float64 `json:"OverallRequestsRate"`
	OverallSentBytesPerSecond     float64 `json:"OverallSentBytesPerSecond"`
	OverallReceivedBytesPerSecond float64 `json:"OverallReceivedBytesPerSecond"`

	HTTPResponseStatuses map[string]uint64 `json:"HTTPResponseStatuses"`

	FullRequestDurationSecondsMean      float64            `json:"FullRequestDurationSecondsMean"`
	FullRequestDurationSecondsQuantiles map[string]float64 `json:"FullRequestDurationSecondsQuantiles"`

	DNSRequests                 uint64             `json:"DNSRequests"`
	DNSDurationSecondsMean      float64            `json:"DNSDurationSecondsMean"`
	DNSDurationSecondsQuantiles map[string]float64 `json:"DNSDurationSecondsQuantiles"`

	TCPConnections              uint64             `json:"TCPConnections"`
	TCPDurationSecondsMean      float64            `json:"TCPDurationSecondsMean"`
	TCPDurationSecondsQuantiles map[string]float64 `json:"TCPDurationSecondsQuantiles"`

	TLSHandshakes               uint64             `json:"TLSHandshakes"`
	TLSDurationSecondsMean      float64            `json:"TLSDurationSecondsMean"`
	TLSDurationSecondsQuantiles map[string]float64 `json:"TLSDurationSecondsQuantiles"`

	HTTPWriteRequestBodyDurationSecondsMean      float64            `json:"HTTPWriteRequestBodyDurationSecondsMean"`
	HTTPWriteRequestBodyDurationSecondsQuantiles map[string]float64 `json:"HTTPWriteRequestBodyDurationSecondsQuantiles"`

	HTTPTimeToFirstByteSecondsMean      float64            `json:"HTTPTimeToFirstByteSecondsMean"`
	HTTPTimeToFirstByteSecondsQuantiles map[string]float64 `json:"HTTPTimeToFirstByteSecondsQuantiles"`

	HTTPResponseDurationSecondsMean      float64            `json:"HTTPResponseDurationSecondsMean"`
	HTTPResponseDurationSecondsQuantiles map[string]float64 `json:"HTTPResponseDurationSecondsQuantiles"`
}

// Get report
func report(config appConfig, duration float64) {
	report := ""

	switch config.report {
	case "json":
		report = reportJson(config, duration)
	default:
		report = reportTextOld(config, duration)
	}

	fmt.Print(report)
}

// Get report struct
func collectReport(config appConfig, duration float64) appReport {

	report := appReport{}

	// Make a copy of main labels map
	statusLabels := make(map[string]string, 0)
	for k, v := range config.metrics.labels {
		statusLabels[k] = v
	}

	report.Target = config.sendEndpoint
	report.SendMethod = config.sendMethod
	report.DurationSeconds = duration
	report.MaxConcurrency = config.workers
	report.RequestBodySize = int64(len(config.sendPayload))

	// Main results
	report.RequestsCompleted, _ = getCounter(config.metrics.requestsSendCount, config.metrics.labelValues...)
	report.RequestsSucceeded, _ = getCounter(config.metrics.responseBytesCount, config.metrics.labelValues...)
	report.RequestsFailed, _ = getCounter(config.metrics.requestsSendErrors, config.metrics.labelValues...)

	if requests, err := getCounter(config.metrics.requestsSendCount, config.metrics.labelValues...); err == nil {
		report.OverallRequestsRate = requests / duration
	}

	// Time per request and transfer rates
	if _, seconds, err := getCountSumFromSummary(registry, "minigun_requests_duration_seconds", config.metrics.labels); err == nil {

		bytes, _ := getCounter(config.metrics.requestsSendBytesSum, config.metrics.labelValues...)
		report.OverallSentBytesPerSecond = bytes / seconds

		bytes, _ = getCounter(config.metrics.responseBytesSum, config.metrics.labelValues...)
		report.OverallReceivedBytesPerSecond = bytes / seconds

		// DNS info
		if requests, _, mean, quantiles, err := getSummaryValues(registry, "minigun_httptrace_dns_duration_seconds", config.metrics.labels); err == nil && requests > 0 {
			report.DNSRequests = requests
			report.DNSDurationSecondsMean = mean
			report.DNSDurationSecondsQuantiles = jsonizeFloatMap(quantiles)
		}

		// TCP connection info
		if requests, _, mean, quantiles, err := getSummaryValues(registry, "minigun_httptrace_connect_duration_seconds", config.metrics.labels); err == nil {
			report.TCPConnections = requests
			report.TCPDurationSecondsMean = mean
			report.TCPDurationSecondsQuantiles = jsonizeFloatMap(quantiles)
		}

		// TLS info
		if requests, _, mean, quantiles, err := getSummaryValues(registry, "minigun_httptrace_tls_handshake_duration_seconds", config.metrics.labels); err == nil && requests > 0 {
			report.TLSHandshakes = requests
			report.TLSDurationSecondsMean = mean
			report.TLSDurationSecondsQuantiles = jsonizeFloatMap(quantiles)
		}

		// Get HTTP statuses info
		statuses, err := getSummaryLabelValues(registry, "minigun_response_duration_seconds", "status")
		if err == nil {
			report.HTTPResponseStatuses = make(map[string]uint64)

			for _, status := range statuses {
				statusLabels["status"] = status
				if statusCount, _, err := getCountSumFromSummary(registry, "minigun_response_duration_seconds", statusLabels); err == nil {
					report.HTTPResponseStatuses[status] = statusCount
				}
			}
		}

		// Get more quantiles
		if _, _, mean, quantiles, err := getSummaryValues(registry, "minigun_requests_duration_seconds", config.metrics.labels); err == nil {
			report.FullRequestDurationSecondsMean = mean
			report.FullRequestDurationSecondsQuantiles = jsonizeFloatMap(quantiles)
		}

		if _, _, mean, quantiles, err := getSummaryValues(registry, "minigun_response_duration_seconds", config.metrics.labels); err == nil {
			report.HTTPResponseDurationSecondsMean = mean
			report.HTTPResponseDurationSecondsQuantiles = jsonizeFloatMap(quantiles)
		}

		if _, _, mean, quantiles, err := getSummaryValues(registry, "minigun_httptrace_write_request_body_duration_seconds", config.metrics.labels); err == nil {
			report.HTTPWriteRequestBodyDurationSecondsMean = mean
			report.HTTPWriteRequestBodyDurationSecondsQuantiles = jsonizeFloatMap(quantiles)
		}

		if _, _, mean, quantiles, err := getSummaryValues(registry, "minigun_httptrace_time_to_first_byte_seconds", config.metrics.labels); err == nil {
			report.HTTPTimeToFirstByteSecondsMean = mean
			report.HTTPTimeToFirstByteSecondsQuantiles = jsonizeFloatMap(quantiles)
		}
	}

	return report
}

// Helper func which converts float64 map keys to string, JSON supports only strings as map keys
func jsonizeFloatMap(in map[float64]float64) map[string]float64 {
	result := make(map[string]float64)

	for k, v := range in {
		result[fmt.Sprintf("%v", k)] = v
	}

	return result
}

// Get json report
func reportJson(config appConfig, duration float64) string {
	var jsonReport []byte
	var err error

	report := collectReport(config, duration)

	if config.prettyJson {
		jsonReport, err = json.MarshalIndent(report, "", "  ")
	} else {
		jsonReport, err = json.Marshal(report)
	}

	if err != nil {
		applog.Errorf("Failed to JSON marshal report: %s", err.Error())
		return "{}"
	}

	return fmt.Sprint(string(jsonReport))
}

// Formats a table into a printable string
func formatPrintMatrix(header printRow, matrix printMatrix, printHeder, borders bool) string {
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)

	if printHeder {
		table.SetHeader(header)
		table.SetAutoFormatHeaders(true)
	}

	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	if !borders {
		table.SetHeaderLine(false)
		table.SetBorder(false)
		table.SetAutoWrapText(false)
		table.SetCenterSeparator("")
		table.SetColumnSeparator("")
		table.SetRowSeparator("")
		table.SetTablePadding("    ")
		table.SetNoWhiteSpace(true)
	} else {
		table.SetRowLine(true)
	}

	table.AppendBulk(matrix)
	table.Render()

	return fmt.Sprintln(tableString.String())
}

// Get human readable text report
func reportTextOld(config appConfig, duration float64) string {
	var outMatrix, outLatencies printMatrix
	var outHeader printRow

	reportBorders := config.report == "table"

	// Make a copy of main labels map
	statusLabels := make(map[string]string, 0)
	for k, v := range config.metrics.labels {
		statusLabels[k] = v
	}

	fmt.Println()

	// Main benchmark info
	outMatrix = append(outMatrix, printRow{"Target:", config.sendEndpoint})
	outMatrix = append(outMatrix, printRow{"Method:", config.sendMethod})
	outMatrix = append(outMatrix, printRow{"Duration:", fmt.Sprintf("%.2f seconds", duration)})
	outMatrix = append(outMatrix, printRow{"Max concurrency:", fmt.Sprintf("%v", config.workers)})
	outMatrix = append(outMatrix, printRow{"Request body size:", fmt.Sprintf("%v", humanizeBytes(int64(len(config.sendPayload)), false))})

	// Separator
	if reportBorders {
		outMatrix = append(outMatrix, printRow{"", ""})
	}

	// Main results
	outMatrix = addCounterToReport(outMatrix, "Completed requests:", config.metrics.requestsSendCount, config.metrics.labelValues...)
	outMatrix = addCounterToReport(outMatrix, "Succeeded requests:", config.metrics.responseBytesCount, config.metrics.labelValues...)
	outMatrix = addCounterToReport(outMatrix, "Failed requests:", config.metrics.requestsSendErrors, config.metrics.labelValues...)

	if requests, err := getCounter(config.metrics.requestsSendCount, config.metrics.labelValues...); err == nil {
		rate := requests / duration
		outMatrix = append(outMatrix, printRow{"Requests per second:", fmt.Sprintf("%.2f (mean, across all concurrent requests)", rate)})

		if config.abTimePerRequest {
			// Mean request time
			if _, _, mean, _, err := getSummaryValues(registry, "minigun_requests_duration_seconds", config.metrics.labels); err == nil {
				outMatrix = append(outMatrix, printRow{"Time per request", fmt.Sprintf("%v (mean)", humanizeDurationSeconds(mean))})
			}
			// Request time over all, cross all the concurency workers
			overallRequestTime := humanizeDurationSeconds(duration / float64(requests))
			outMatrix = append(outMatrix, printRow{"Time per request", fmt.Sprintf("%v (mean, across all concurrent requests)", overallRequestTime)})
		}
	}

	// Time per request and transfer rates
	if _, seconds, err := getCountSumFromSummary(registry, "minigun_requests_duration_seconds", config.metrics.labels); err == nil {

		if bytes, err := getCounter(config.metrics.requestsSendBytesSum, config.metrics.labelValues...); err == nil {
			tmpPrint := fmt.Sprintf("%v sent (mean)\n%v sent (mean, across all concurrent requests)",
				humanizeBytes(int64(bytes/seconds), true), humanizeBytes(int64(bytes/duration), true))

			if bytes, err := getCounter(config.metrics.responseBytesSum, config.metrics.labelValues...); err == nil {
				tmpPrint += fmt.Sprintf("\n%v received (mean)\n%v received (mean, across all concurrent requests)",
					humanizeBytes(int64(bytes/seconds), true), humanizeBytes(int64(bytes/duration), true))
			}

			outMatrix = append(outMatrix, printRow{"Transfer rate (HTTP Message Body)", tmpPrint})
		}

		// DNS info
		if requests, _, err := getCountSumFromSummary(registry, "minigun_httptrace_dns_duration_seconds", config.metrics.labels); err == nil && requests > 0 {
			outMatrix = append(outMatrix, printRow{"DNS queries", fmt.Sprintf("%v", requests)})
		}

		// TCP connection info
		if requests, _, err := getCountSumFromSummary(registry, "minigun_httptrace_connect_duration_seconds", config.metrics.labels); err == nil {
			outMatrix = append(outMatrix, printRow{"TCP connections", fmt.Sprintf("%v", requests)})
		}

		// TLS info
		if requests, _, err := getCountSumFromSummary(registry, "minigun_httptrace_tls_handshake_duration_seconds", config.metrics.labels); err == nil && requests > 0 {
			outMatrix = append(outMatrix, printRow{"TLS Handshakes", fmt.Sprintf("%v", requests)})
		}

		// Get HTTP statuses info
		statuses, err := getSummaryLabelValues(registry, "minigun_response_duration_seconds", "status")
		if err == nil {
			statusReport := make([]string, 0)

			for _, status := range statuses {
				statusLabels["status"] = status
				if statusCount, _, err := getCountSumFromSummary(registry, "minigun_response_duration_seconds", statusLabels); err == nil {
					statusReport = append(statusReport, fmt.Sprintf("[%v:%v]", status, statusCount))
				}
			}

			if len(statusReport) > 0 {
				outMatrix = append(outMatrix, printRow{"HTTP status codes", fmt.Sprintf("%s", strings.Join(statusReport, " "))})
			}
		}
	}

	// Add the first table to the report
	report := formatPrintMatrix(outHeader, outMatrix, false, reportBorders)

	// Print new table with latencies
	outHeader = printRow{"", "Mean", "Median", "P90", "P95", "P99"}

	// Add latencies based on Prometheus summaries
	outLatencies = addSummaryToReport(outLatencies, "Full request duration", registry, "minigun_requests_duration_seconds", config.metrics.labels)
	outLatencies = addSummaryToReport(outLatencies, "DNS request duration", registry, "minigun_httptrace_dns_duration_seconds", config.metrics.labels)
	outLatencies = addSummaryToReport(outLatencies, "TCP connection duration", registry, "minigun_httptrace_connect_duration_seconds", config.metrics.labels)
	outLatencies = addSummaryToReport(outLatencies, "TLS handshake duration", registry, "minigun_httptrace_tls_handshake_duration_seconds", config.metrics.labels)
	outLatencies = addSummaryToReport(outLatencies, "HTTP write request body", registry, "minigun_httptrace_write_request_body_duration_seconds", config.metrics.labels)
	outLatencies = addSummaryToReport(outLatencies, "HTTP time to first byte", registry, "minigun_httptrace_time_to_first_byte_seconds", config.metrics.labels)
	outLatencies = addSummaryToReport(outLatencies, "HTTP response duration", registry, "minigun_response_duration_seconds", config.metrics.labels)

	// Add the second table to the report
	report += formatPrintMatrix(outHeader, outLatencies, true, reportBorders)

	return report
}
