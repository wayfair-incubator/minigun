package main

func helpReport() string {
	var outMatrix printMatrix
	var outHeader printRow

	// Start help report
	report := `
Minigun report help.

There're some metrics with "(mean, across all concurrent requests)" which show different numbers
than the same metrics with just "(mean)". The "across all concurrent requests" comment means it's
an average number for the entire benchmark run, including and across all benchmark workers.
While metrics with just "(mean)" is an average of per worker metrics.

Let's take a look at "Transfer rate (HTTP Message Body)" metric for example:

"552 kB/s sent (mean)" - every worker calculates number of bytes of HTTP body it sends and exact
time required to send them. So this metric shows average Bytes / second for all workers.

"10 kB/s sent (mean, across all concurrent requests)" - this is average across all workers and
for the entire benchmark duration. We're sending 1.0 kB request body at 10 requests/s rate which
is in total 10 kB/s across all concurrent requests during benchmark duration.
`

	// Main benchmark info
	outHeader = printRow{"METRIC", "EXPLANATION"}
	outMatrix = append(outMatrix, printRow{"Full request duration", "Full time of a request starting from the beginning (DNS lookup) and ending with receiving a full response."})
	outMatrix = append(outMatrix, printRow{"DNS request duration", "The time spent on DNS lookup."})
	outMatrix = append(outMatrix, printRow{"TCP connection duration", "The time spent on establishing TCP connection using a TCP handshake."})
	outMatrix = append(outMatrix, printRow{"TLS handshake duration", "The time spent on TLS handshake."})
	outMatrix = append(outMatrix, printRow{"HTTP write request body", "The time required to write request body to the remote endpoint."})
	outMatrix = append(outMatrix, printRow{"HTTP time to first byte", "The time since the request start and when the first byte of HTTP reply from the remote endpoint is received. This time includes DNS lookup, establishing the TCP connection and SSL handshake if the request is made over https."})
	outMatrix = append(outMatrix, printRow{"HTTP response duration", "The time since request headers and body are sent and until the full response is received."})

	report += "\n\n" + formatPrintMatrix(outHeader, outMatrix, true, false)

	return report
}
