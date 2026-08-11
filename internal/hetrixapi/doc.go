// Package hetrixtools provides a semantic Go client for HetrixTools APIs.
//
// This project is not affiliated with, endorsed by, or supported by
// HetrixTools. It is provided without any guarantee that the HetrixTools API or
// this client will behave as expected for your account.
//
// The package exposes methods named after HetrixTools actions and resources,
// while API version selection, bearer authentication, token-in-path behavior,
// response-shape normalization, and rate-limit handling remain internal to the
// client.
//
// # API references
//
// The client is built from the public HetrixTools API documentation:
//
//   - API overview and versioning: https://docs.hetrixtools.com/understanding-our-apis/
//   - API v3 reference: https://docs.hetrixtools.com/api-v3/
//   - v2 Add website, ping, service, and SMTP uptime monitors:
//     https://docs.hetrixtools.com/api-add-website-ping-service-smtp-uptime-monitor/
//   - v2 Add server-agent heartbeat uptime monitors:
//     https://docs.hetrixtools.com/api-add-server-agent-uptime-monitor-heartbeat-uptime-monitor/
//   - v2 Delete uptime monitors:
//     https://docs.hetrixtools.com/api-delete-uptime-monitor/
//   - v2 Add blacklist monitors:
//     https://docs.hetrixtools.com/api-add-blacklist-monitor/
//   - v2 Edit blacklist monitors:
//     https://docs.hetrixtools.com/api-edit-blacklist-monitor/
//   - v2 Delete blacklist monitors:
//     https://docs.hetrixtools.com/api-delete-blacklist-monitor/
//
// Some uptime and blacklist monitor mutations still use legacy token-path v2
// endpoints. The client exposes canonical Go types and converts them to the API
// shape required by the endpoint being called. For example, UptimeMonitorRequest
// uses canonical Type values such as "http" and canonical Locations values such
// as "new_york", then MarshalJSON converts them to the documented v2 uptime
// payload fields such as Type: 1 and Locations: {"nyc": true}.
//
// # Rate limits and tests
//
// The client owns request pacing, retry, and rate-limit reset handling. The
// behavior is based on the HetrixTools API overview and v3 reference:
//
//   - https://docs.hetrixtools.com/understanding-our-apis/
//   - https://docs.hetrixtools.com/api-v3/
//
// By default, the client waits at least 500ms between requests in each limiter
// bucket. Legacy v2 token-path requests share one limiter. v3 REST requests use
// both a user-level limiter and a normalized method/path endpoint limiter. HTTP
// 429 responses are retried and Retry-After or HetrixTools rate-limit reset
// headers are honored when present.
//
// Tests and examples should pass WithMinimumRequestInterval(0) and use
// httptest.Server or WithHTTPClient so package examples run without external
// network access. Do not disable pacing for production clients unless the caller
// implements equivalent rate limiting.
package hetrixtools
