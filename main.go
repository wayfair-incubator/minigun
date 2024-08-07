// Simple HTTP benchmark tool
//
// @authors Minigun Maintainers
// @copyright 2020 Wayfair, LLC -- All rights reserved.

package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/http2"

	"github.com/dustin/go-humanize"
	"github.com/google/logger"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Constants and vars
const version = "0.6.0"
const workersCannelSize = 1024
const errorBadHTTPCode = "Bad HTTP status code"

var applog *logger.Logger
var workerStatuses []workerStatus
var registry = prometheus.NewRegistry()

// Let's use the same buckets for histograms as NGINX Ingress controller
var secondsDurationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// Objectives for summary distributions
var summaryObjectives = map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001}

// Config is the main app config struct
type appConfig struct {
	workers  int
	verbose  bool
	insecure bool
	instance string
	name     string

	abTimePerRequest bool

	pushGateway  string
	pushInterval time.Duration

	sendMode              string
	sendEndpoint          string
	sendMethod            string
	sendFile              string
	sendTimeout           time.Duration
	sendDisableKeepAlives bool
	sendJSON              bool
	sendPayload           []byte
	sendHTTPHeaders       httpHeaders
	sendBodySize          uint64

	fireDuration time.Duration
	fireRate     int

	report     string
	prettyJson bool

	metrics appMetrics
}

// Client stores pointers to configured remote endpoint writes/clients
type senderClient struct {
	httpClient *http.Client

	socketConn   net.Conn
	socketWriter *bufio.Writer
}

// Message that is sent to workers
type message struct {
	number int
}

// Workers status
type workerStatus struct {
	ID      int  `json:"id"`
	Running bool `json:"running"`
}

// Status defines status
type appStatus struct {
	Workers []workerStatus `json:"workers"`
	Version string         `json:"version"`
}

// Custom type to parse HTTP headers as multiple flag cli args
type httpHeaders map[string]string

func (headers *httpHeaders) String() string {
	return "HTTP headers"
}

func (headers *httpHeaders) Set(value string) error {
	var tmpHeaders httpHeaders

	s := strings.Split(value, ":")
	if len(s) < 2 {
		return fmt.Errorf("wrong header argument")
	}

	if len(*headers) < 1 {
		tmpHeaders = make(httpHeaders)
	} else {
		tmpHeaders = *headers
	}

	tmpHeader := strings.TrimSpace(s[0])
	tmpValue := strings.TrimSpace(strings.Join(s[1:], ":"))

	tmpHeaders[tmpHeader] = tmpValue
	*headers = tmpHeaders

	return nil
}

// Status for future web endpoint
func status(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	myStatus := appStatus{
		Workers: workerStatuses,
		Version: version,
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Make json output
	jsonOut, err := json.Marshal(myStatus)
	applog.Infof("Sending status: %v", myStatus)
	if err != nil {
		applog.Errorf("Failed to json.Marshal() status: %v", err)
		http.Error(w, "Failed to json.Marshal() status", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(jsonOut))
}

// Prometheus metrics handler
func metrics(w http.ResponseWriter, r *http.Request) {
	applog.V(8).Info("Got HTTP request for /metrics")

	promhttp.HandlerFor(prometheus.Gatherer(registry), promhttp.HandlerOpts{}).ServeHTTP(w, r)
}

// Health-check handler
func health(w http.ResponseWriter, r *http.Request) {
	applog.V(8).Info("Got HTTP request for /health")
	healthy := true

	for id, status := range workerStatuses {
		if !status.Running {
			healthy = false
			applog.V(8).Infof("Worker %v is not running", id)
		}
	}

	if healthy {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "All workers are up and running")
		return
	}

	http.Error(w, "Some workers are not running. Check applog for more details", http.StatusInternalServerError)
}

// Main web server
func runMainWebServer(listen string) {
	// Setup http router
	router := mux.NewRouter().StrictSlash(true)

	// Prometheus metrics
	router.HandleFunc("/metrics", metrics).Methods("GET")

	// Health-check endpoint
	router.HandleFunc("/health", health).Methods("GET")

	// Status endpoint
	router.HandleFunc("/status", status).Methods("GET")

	// Log
	applog.Info("Main web server started")

	// Run main http router
	applog.Fatal(http.ListenAndServe(listen, router))
}

// Random string generator
func randomBytes(size uint64) []byte {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	rand.Seed(time.Now().UnixNano())

	result := make([]byte, size)
	for i := range result {
		result[i] = letters[rand.Intn(len(letters))]
	}

	return result
}

// Init client
func initClient(config appConfig) (senderClient, error) {
	var err error
	client := senderClient{}

	switch config.sendMode {

	case "http":
		tr := &http.Transport{
			DisableKeepAlives: config.sendDisableKeepAlives,
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: config.insecure}}
		client.httpClient = &http.Client{Transport: tr, Timeout: config.sendTimeout}

	case "http2":
		tr := &http2.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: config.insecure}}
		client.httpClient = &http.Client{Transport: tr, Timeout: config.sendTimeout}

	case "socket":
		client.socketConn, err = net.DialTimeout(config.sendMethod, config.sendEndpoint, config.sendTimeout)
		if err == nil {
			client.socketWriter = bufio.NewWriter(client.socketConn)
		}

	default:
		err = fmt.Errorf("unsupported sendMode")
	}

	return client, err
}

// Close client
func closeClient(config appConfig, client senderClient) error {
	var err error

	switch config.sendMode {
	case "http", "http2":
		client.httpClient.CloseIdleConnections()
	case "socket":
		err = client.socketConn.Close()
	default:
		err = fmt.Errorf("unsupported sendMode")
	}

	return err
}

// Send data via HTTP
func sendDataHTTP(data []byte, config appConfig, client *http.Client) error {
	var start, wroteRequest, connect, headers, dns, tlsHandshake time.Time

	req, err := http.NewRequest(config.sendMethod, config.sendEndpoint, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	if config.sendJSON {
		req.Header.Set("Content-Type", "application/json")
	} else {
		req.Header.Set("Content-Type", "text/plain")
	}

	for key, value := range config.sendHTTPHeaders {
		if key == "Host" {
			req.Host = value
		} else {
			req.Header.Set(key, value)
		}
	}

	applog.Infof("Sending %v bytes to %s", len(data), config.sendEndpoint)

	// HTTP trace
	trace := &httptrace.ClientTrace{
		DNSStart: func(dsi httptrace.DNSStartInfo) { dns = time.Now() },
		DNSDone: func(ddi httptrace.DNSDoneInfo) {
			config.metrics.histDNSDuration.WithLabelValues(config.metrics.labelValues...).Observe(time.Since(dns).Seconds())
			config.metrics.summaryDNSDuration.WithLabelValues(config.metrics.labelValues...).Observe(time.Since(dns).Seconds())
			applog.Infof("DNS Done: %v\n", time.Since(dns))
			applog.Infof("DNS Result: %v\n", ddi.Addrs)
		},

		TLSHandshakeStart: func() { tlsHandshake = time.Now() },
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			config.metrics.histTLSHandshakeDuration.WithLabelValues(config.metrics.labelValues...).Observe(time.Since(tlsHandshake).Seconds())
			config.metrics.summaryTLSHandshakeDuration.WithLabelValues(config.metrics.labelValues...).Observe(time.Since(tlsHandshake).Seconds())
			applog.Infof("TLS Handshake: %v\n", time.Since(tlsHandshake))
		},

		ConnectStart: func(network, addr string) { connect = time.Now() },
		ConnectDone: func(network, addr string, err error) {
			config.metrics.histConnectDuration.WithLabelValues(config.metrics.labelValues...).Observe(time.Since(connect).Seconds())
			config.metrics.summaryConnectDuration.WithLabelValues(config.metrics.labelValues...).Observe(time.Since(connect).Seconds())
			applog.Infof("Connect time: %v\n", time.Since(connect))
		},

		WroteHeaders: func() { headers = time.Now() },
		WroteRequest: func(wri httptrace.WroteRequestInfo) {
			config.metrics.histWroteRequestBodyDuration.WithLabelValues(config.metrics.labelValues...).Observe(time.Since(headers).Seconds())
			config.metrics.summaryWroteRequestBodyDuration.WithLabelValues(config.metrics.labelValues...).Observe(time.Since(headers).Seconds())
			wroteRequest = time.Now()
			applog.Infof("Wrote request body time: %v\n", time.Since(headers))
		},

		GotFirstResponseByte: func() {
			config.metrics.histGotFirstByteDuration.WithLabelValues(config.metrics.labelValues...).Observe(time.Since(start).Seconds())
			config.metrics.summaryGotFirstByteDuration.WithLabelValues(config.metrics.labelValues...).Observe(time.Since(start).Seconds())
			applog.Infof("Time from start to first byte: %v\n", time.Since(start))
		},
	}

	// Do request
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	start = time.Now()
	resp, err := client.Do(req)
	if !wroteRequest.IsZero() {
		responseTime := time.Since(wroteRequest)
		if err == nil && resp != nil {
			localLabelValues := append(config.metrics.labelValues, fmt.Sprintf("%v", resp.StatusCode))
			config.metrics.histResponseDuration.WithLabelValues(localLabelValues...).Observe(responseTime.Seconds())
			config.metrics.summaryResponseDuration.WithLabelValues(localLabelValues...).Observe(responseTime.Seconds())
		}
	}

	totalTime := time.Since(start)

	config.metrics.requestsSendBytesSum.WithLabelValues(config.metrics.labelValues...).Add(float64(len(config.sendPayload)))
	config.metrics.histRequestsDuration.WithLabelValues(config.metrics.labelValues...).Observe(totalTime.Seconds())
	config.metrics.summaryRequestsDuration.WithLabelValues(config.metrics.labelValues...).Observe(totalTime.Seconds())

	applog.Infof("Total time: %v\n", totalTime)

	if err != nil {
		applog.Errorf("Failed to send data to %q, error: %s", config.sendEndpoint, err.Error())
		return err
	}

	defer resp.Body.Close()

	applog.Infof("Data sent to %s using %s method, response status %v", config.sendEndpoint, config.sendMethod, resp.StatusCode)

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			applog.Errorf("Failed to read response body from %q, error: %s", config.sendEndpoint, err.Error())
			return err
		}

		config.metrics.responseBytesCount.WithLabelValues(config.metrics.labelValues...).Inc()
		config.metrics.responseBytesSum.WithLabelValues(config.metrics.labelValues...).Add(float64(len(bodyBytes)))
		return nil
	}

	// If we're here, then response status was not good
	return fmt.Errorf(errorBadHTTPCode)
}

// Send data via socket
func sendDataSocket(data []byte, writer *bufio.Writer) error {
	number, err := writer.Write(data)
	if err == nil {
		err = writer.Flush()
		applog.Infof("Successfully sent %v bytes", number)
	}

	return err
}

// Send data to a remote endpoint
func sendData(data []byte, config appConfig, client senderClient) error {

	switch config.sendMode {

	case "http", "http2":
		return sendDataHTTP(data, config, client.httpClient)

	case "socket":
		return sendDataSocket(data, client.socketWriter)

	default:
		return fmt.Errorf("unsupported send mode: %s", config.sendMode)
	}
}

// Main benchmark loop
func fire(ctx context.Context, config appConfig, comm *chan message) {
	var nanoseconds int32

	if config.fireRate > 0 {
		nanoseconds = int32(float64(1) / float64(config.fireRate) * 1000000000)
	} else {
		nanoseconds = 1
	}
	duration := time.Duration(nanoseconds) * time.Nanosecond
	tick := time.Tick(duration)

	applog.Infof("Rate: %v", config.fireRate)
	applog.Infof("Tick nanoseconds: %v", nanoseconds)
	applog.Infof("Tick duration: %v", duration)

	// Keep fireing until we receive exit signal
	for {
		select {
		// Exit signal
		case <-ctx.Done():
			applog.Info("Fire function exiting")
			close(*comm)
			return
		// Tick event
		case <-tick:
			if len(*comm) < workersCannelSize {
				*comm <- message{number: 1}
			} else {
				config.metrics.channelFullEvents.WithLabelValues(config.metrics.labelValues...).Inc()
			}
		}
	}

}

// Metrics updater
func updateMetrics(config appConfig, comm *chan message) {
	// Updating every 2 seconds is frequent enough
	tick := time.Tick(2 * time.Second)

	for {
		select {
		// Tick handler
		case <-tick:
			config.metrics.channelLength.WithLabelValues(config.metrics.labelValues...).Set(float64(len(*comm)))
		}
	}
}

// Worker
func worker(ctx context.Context, id int, config appConfig, comm chan message, status *workerStatus, wg *sync.WaitGroup) {

	applog.Infof("Worker %d started", id)
	defer wg.Done()
	status.ID = id
	status.Running = true

	// Init client per worker to use keep alive where possible
	client, err := initClient(config)
	if err != nil {
		status.Running = false
		applog.Errorf("Worker %v: Failed to initialize sender client: %s", id, err.Error())
		applog.Errorf("Worker %v failed, exiting", id)
		return
	}

	// Main select
	for {
		select {

		case <-ctx.Done():
			status.Running = false

			err = closeClient(config, client)
			if err != nil {
				applog.Errorf("Worker %v: Error closing sender client: %s", id, err.Error())
			}

			applog.Infof("Worker %d exiting", id)
			return

		case <-comm:

			applog.Infof("Worker %d: processing task", id)

			err := sendData(config.sendPayload, config, client)
			config.metrics.requestsSendCount.WithLabelValues(config.metrics.labelValues...).Inc()

			if err != nil {

				config.metrics.requestsSendErrors.WithLabelValues(config.metrics.labelValues...).Inc()

				if err.Error() != errorBadHTTPCode {

					// Sending failed and it's not because of HTTP code, let's try to reconnect
					applog.Infof("Worker %v: Re-establishing connection", id)
					err := closeClient(config, client)
					if err != nil {
						applog.Infof("Worker %v: Error closing sender client: %s", id, err.Error())
					}

					client, err = initClient(config)
					if err != nil {
						status.Running = false
						applog.Errorf("Worker %v: Failed to initialize sender client: %s", id, err.Error())
						applog.Errorf("Worker %v failed, exiting", id)
						return
					}
				}

			} else {
				config.metrics.requestsSendSuccess.WithLabelValues(config.metrics.labelValues...).Inc()
			}
		}
	}
}

// Convert bytes to a human readable format
func humanizeBytes(b int64, rate bool) string {
	var suffix string

	if rate {
		suffix = "/s"
	}

	return fmt.Sprintf("%s%s",
		humanize.Bytes(uint64(b)),
		suffix)
}

// Returns human readable duration for value in float64 seconds
func humanizeDurationSeconds(seconds float64) string {
	const unit = 1000
	const nanosecondsInSecond = 1e+9

	duration := seconds * nanosecondsInSecond

	if duration < unit {
		return fmt.Sprintf("%.2fns", duration)
	}

	div, exp := int64(unit), 0
	units := [3]string{"µs", "ms", "s"}

	for n := duration / unit; n >= unit; n /= unit {
		div *= unit
		exp++
		if exp >= len(units)-1 {
			break
		}
	}

	return fmt.Sprintf("%.2f%s",
		float64(duration)/float64(div), units[exp])
}

// Functions for pushing metrics
func prometheusMetricsPusher(config appConfig) {
	tick := time.Tick(config.pushInterval)

	pusher := push.New(config.pushGateway, "minigun").Gatherer(registry)

	for {
		select {
		// Tick event
		case <-tick:

			applog.Info("Pushing metrics to Prometheus Pushgateway")

			if err := pusher.Add(); err != nil {
				applog.Errorf("Could not push to Pushgateway: %s", err.Error())
			}
		}
	}

}

// Prints info to stdout or stderr
func info(config appConfig, line string) {
	if config.report == "text" || config.report == "table" {
		fmt.Printf("%s\n", line)
	}
}

// Check URL
func validateUrl(inURL string) error {

	u, err := url.Parse(inURL)

	if err != nil {
		return err
	}

	if u.Scheme == "" {
		return fmt.Errorf("can't find scheme in URL %q", inURL)
	}

	if u.Host == "" {
		return fmt.Errorf("can't find host in URL %q", inURL)
	}

	return nil
}

// Main!
func main() {
	var listen, randomBodySize string
	var wg sync.WaitGroup
	var showVersion, explainReport bool

	// Init config
	config := appConfig{}

	//Make a background context.
	ctx := context.Background()
	// Make a new context with cancel, we'll use it to make sure all routines can exit properly.
	ctxWithCancel, cancelFunction := context.WithCancel(ctx)

	// Arguments
	flag.BoolVar(&showVersion, "version", false, "Show version and exit")
	flag.BoolVar(&explainReport, "report-help", false, "Show detailed explanation of reported metrics and exit")

	flag.StringVar(&config.sendEndpoint, "fire-target", "", "Benchmark target endpoint")
	flag.DurationVar(&config.fireDuration, "fire-duration", time.Second*10, "Duration of the benchmark. Specify 0 to run forever until stopped")
	flag.IntVar(&config.fireRate, "fire-rate", 0, "Desired rate in requests/sec. Default is 0 - unlimited")

	flag.IntVar(&config.workers, "workers", 1, "The number of worker threads")
	flag.BoolVar(&config.verbose, "verbose", false, "Print INFO level applog to stdout")
	flag.BoolVar(&config.insecure, "insecure", false, "Ignore TLS certificate errors")
	flag.StringVar(&config.sendMethod, "send-method", "GET", "Send method, like GET, POST, PUT, etc")
	flag.DurationVar(&config.sendTimeout, "send-timeout", time.Second*5, "Send request timeout")
	flag.BoolVar(&config.sendDisableKeepAlives, "disable-keep-alive", false, "Disable HTTP KeepAlive when sending")
	flag.BoolVar(&config.sendJSON, "send-json", true, "Send JSON encoded or plain text. Works with HTTP only")
	flag.StringVar(&config.sendFile, "send-file", "", "Send contents of this file")
	flag.StringVar(&listen, "listen", ":8765", "Address:port to listen on for exposing metrics")
	flag.Var(&config.sendHTTPHeaders, "http-header", "Custom HTTP header in 'Header:Value' form. Can be specified multiple times")
	flag.StringVar(&randomBodySize, "random-body-size", "", "Generate random number of bytes and send them as HTTP message body. Example: 1KB")

	flag.StringVar(&config.report, "report", "text", "Report format. One of: 'text', 'table', 'json'")
	flag.BoolVar(&config.prettyJson, "pretty-json", false, "Pretty print JSON report with indents")

	flag.BoolVar(&config.abTimePerRequest, "ab-time-per-request", false, "Show Apache Benchmark style time per request metric")

	flag.StringVar(&config.name, "name", "default", "Benchmark run name. It will be used as 'name' label for metrics. Can be used for grouping all instances.")
	flag.StringVar(&config.instance, "instance", "", "Benchmark instance name. It will be used as 'instance' label for metrics. Default to hostname.")
	flag.StringVar(&config.pushGateway, "push-gateway", "", "Prometheus Pushgateway URL")
	flag.DurationVar(&config.pushInterval, "push-interval", time.Second*15, "Metrics push interval")

	flag.StringVar(&config.sendMode, "send-mode", "http", "Send mode, supported options are http and http2")

	flag.Parse()

	// For now we support http and http2 only
	if config.sendMode != "http" && config.sendMode != "http2" {
		fmt.Printf("Unsuported -send-mode=%q. Only 'http' and 'http2' are supported at the moment\n", config.sendMode)
		os.Exit(1)
	}

	// Show and exit functions
	if showVersion {
		fmt.Printf("Version: %s\n", version)
		os.Exit(0)
	}

	if explainReport {
		fmt.Println(helpReport())
		os.Exit(0)
	}

	if config.instance == "" {
		if hostname, err := os.Hostname(); err == nil {
			config.instance = strings.ToLower(hostname)
		} else {
			applog.Errorf("Failed to get hostaname: %s", err.Error())
			config.instance = "default"
		}
	}

	// Initialize the global status var
	workerStatuses = make([]workerStatus, config.workers)

	// Logger
	applog = logger.Init("minigun", config.verbose, false, io.Discard)

	// Some checks
	if config.sendEndpoint == "" {
		applog.Fatal("-fire-target is not specified")
	} else if err := validateUrl(config.sendEndpoint); err != nil {
		applog.Fatal(err.Error())
	}

	// Convert randomBodySize
	if randomBodySize != "" {
		if parsedSize, err := humanize.ParseBytes(randomBodySize); err == nil {
			config.sendBodySize = parsedSize
		} else {
			applog.Fatalf("Error parsing -random-body-size: %s", err.Error())
		}
	}

	// Load file if requested
	if config.sendFile != "" {
		applog.Infof("Reading file %q", config.sendFile)
		if data, err := os.ReadFile(config.sendFile); err == nil {
			config.sendPayload = data
		} else {
			applog.Fatalf("Error reading file %q: %s", config.sendFile, err.Error())
		}
	} else {
		if config.sendBodySize > 0 {
			applog.Infof("Generating random request body, size: %v", config.sendBodySize)
			config.sendPayload = randomBytes(config.sendBodySize)
		}
	}

	// Push interval sanity check
	if config.pushInterval < 10*time.Second {
		applog.Fatal("-push-interval must be >= 10 seconds")
	}

	info(config, "Starting benchmark")

	// Init metric
	config.metrics = initMetrics(
		config,
		[]string{
			"version",
			"name",
			"instance",
		},
		[]string{
			version,
			config.name,
			config.instance,
		})

	registry.MustRegister()

	// Run a separate routine with http server
	go runMainWebServer(listen)

	// Make a channel and start workers
	comm := make(chan message, workersCannelSize)
	for i := 0; i < config.workers; i++ {
		wg.Add(1)
		go worker(ctxWithCancel, i, config, comm, &workerStatuses[i], &wg)
	}

	// Channels for signal processing and locking main()
	sigs := make(chan os.Signal, 1)
	exit := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Run metrics updater routine
	go updateMetrics(config, &comm)

	// Fire!!!
	timeout := time.After(config.fireDuration)
	started := time.Now()
	go fire(ctxWithCancel, config, &comm)

	// Start metrics pusher if enabled
	if config.pushGateway != "" {
		go prometheusMetricsPusher(config)
	}

	// Exit after duration by sending signal to "exit" channel
	if config.fireDuration > 0 {
		go func() {
			<-timeout
			applog.Info("Fire duration done")
			cancelFunction()
			exit <- true
		}()
	} else {
		info(config, "Running until stopped")
	}

	// Wait for signals to exit and send signal to "exit" channel
	go func() {
		sig := <-sigs
		fmt.Printf("\nReceived signal: %v\n", sig)
		cancelFunction()
		exit <- true
	}()

	info(config, "Benchmark is running.")
	<-exit
	duration := time.Since(started).Seconds()

	// Wait for workers to exit
	wg.Wait()
	info(config, "Benchmark is complete.")

	// Report
	report(config, duration)
}
