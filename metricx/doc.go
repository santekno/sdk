// Package metricx provides a vendor-neutral metrics interface and a default
// in-memory implementation. Users who want Prometheus can wrap their
// prometheus.Counter / prometheus.Histogram in the metricx interfaces.
//
// Standard metric names used by other SDK packages:
//
//	santekno_http_requests_total              (counter)
//	santekno_http_request_duration_seconds    (histogram)
//	santekno_http_requests_in_flight           (gauge)
//	santekno_httpclient_requests_total         (counter)
//	santekno_httpclient_request_duration_seconds (histogram)
//	santekno_httpclient_breaker_state          (gauge)
package metricx
