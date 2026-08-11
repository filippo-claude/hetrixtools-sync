// Modified by the hetrixtools-sync project from the Apache-2.0 upstream source.
package hetrixtools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
)

type (
	// UptimeMonitorRequest describes an uptime monitor create or update request.
	//
	// The client sends this request to the documented HetrixTools v2 uptime add
	// endpoint for creates and edit endpoint for updates:
	//
	//   - Website, ping, service, and SMTP monitors:
	//     https://docs.hetrixtools.com/api-add-website-ping-service-smtp-uptime-monitor/
	//   - Server-agent heartbeat monitors:
	//     https://docs.hetrixtools.com/api-add-server-agent-uptime-monitor-heartbeat-uptime-monitor/
	//   - Uptime monitor edit endpoint:
	//     https://gist.github.com/hetrixtools/b6fed467837874491f2a046fac957293
	//
	// The Go fields are canonicalized for client users. MarshalJSON converts them
	// to the v2 payload shape documented by HetrixTools, including numeric Type
	// values and short monitoring-location codes.
	UptimeMonitorRequest struct {
		// MID is the existing uptime monitor ID for update requests. Leave empty for create requests.
		MID string `json:"MID,omitempty"`
		// Type is the canonical monitor type: http, ping, smtp, or heartbeat.
		Type string `json:"-" validate:"omitempty,oneof=http ping smtp heartbeat"`
		// Name is the human-readable monitor name.
		Name string `json:"Name,omitempty"`
		// Target is the URL, host, IP, or SMTP hostname checked by non-heartbeat monitors.
		Target string `json:"Target,omitempty"`
		// Port is the SMTP port and is only valid for smtp monitors.
		Port int64 `json:"Port,omitempty"`
		// HTTPMethod is the HTTP method and is only valid for http monitors.
		HTTPMethod string `json:"Method,omitempty"`
		// MaxRedirects is the maximum redirects to follow and is only valid for http monitors.
		MaxRedirects int64 `json:"MaxRedirects,omitempty"`
		// SMTPUser is the optional SMTP username and is only valid for smtp monitors.
		SMTPUser string `json:"SMTPUser,omitempty"`
		// SMTPPass is the optional SMTP password and is only valid for smtp monitors.
		SMTPPass string `json:"SMTPPass,omitempty"`
		// Timeout is the check timeout in seconds.
		Timeout int64 `json:"Timeout,omitempty"`
		// Frequency is the check frequency in minutes.
		Frequency int64 `json:"Frequency,omitempty"`
		// FailsBeforeAlert is the number of failed checks required before alerting.
		FailsBeforeAlert int64 `json:"FailsBeforeAlert,omitempty"`
		// FailedLocations is the number of failed locations required before alerting.
		FailedLocations int64 `json:"FailedLocations,omitempty"`
		// ContactList is the HetrixTools contact list ID used for notifications.
		ContactList string `json:"ContactList,omitempty"`
		// Category is the HetrixTools monitor category.
		Category string `json:"Category,omitempty"`
		// AlertAfter is the delay before sending an alert, such as 5m.
		AlertAfter string `json:"AlertAfter,omitempty"`
		// RepeatTimes is the number of times to repeat alerts.
		RepeatTimes int64 `json:"RepeatTimes,omitempty"`
		// RepeatEvery is the alert repeat interval, such as 60m.
		RepeatEvery string `json:"RepeatEvery,omitempty"`
		// Public controls whether the monitor has a public report.
		Public *bool `json:"Public,omitempty"`
		// ShowTarget controls whether the monitor target is shown publicly.
		ShowTarget *bool `json:"ShowTarget,omitempty"`
		// VerSSLCert controls SSL certificate validation for http and smtp monitors.
		VerSSLCert *bool `json:"VerSSLCert,omitempty"`
		// VerSSLHost controls SSL hostname validation for http and smtp monitors.
		VerSSLHost *bool `json:"VerSSLHost,omitempty"`
		// Locations is the set of canonical monitoring location names enabled for the monitor.
		Locations []string `json:"-" validate:"omitempty,dive,uptime_location"`
		// Keyword is the expected response-body keyword and is only valid for http monitors.
		Keyword string `json:"-"`
		// HTTPCodes is the set of accepted HTTP status codes and is only valid for http monitors.
		HTTPCodes []int64 `json:"-"`
		// Grace is the heartbeat grace period and is only valid for heartbeat monitors.
		Grace int64 `json:"Grace,omitempty"`
		// INFOPub controls whether heartbeat info details are public.
		INFOPub *bool `json:"INFOPub,omitempty"`
		// CPUPub controls whether heartbeat CPU details are public.
		CPUPub *bool `json:"CPUPub,omitempty"`
		// RAMPub controls whether heartbeat RAM details are public.
		RAMPub *bool `json:"RAMPub,omitempty"`
		// DISKPub controls whether heartbeat disk details are public.
		DISKPub *bool `json:"DISKPub,omitempty"`
		// NETPub controls whether heartbeat network details are public.
		NETPub *bool `json:"NETPub,omitempty"`
		// SSLExpirationReminder is the number of days before SSL expiration to alert; zero disables it.
		SSLExpirationReminder int64 `json:"SSLExpiryReminder,omitempty"`
		// DomainExpirationReminder is the number of days before domain expiration to alert; zero disables it.
		DomainExpirationReminder int64 `json:"DomainExpiryReminder,omitempty"`
		// NSChangeAlert controls nameserver-change alerts.
		NSChangeAlert *bool `json:"NSChangeAlert,omitempty"`
	}

	// ListUptimeMonitorsRequest filters uptime monitor list results.
	ListUptimeMonitorsRequest struct {
		// PaginationRequest contains page and per_page filters. Uptime monitors accept per_page up to 200.
		PaginationRequest
		// ID filters by uptime monitor ID. HetrixTools returns only that monitor and ignores all other filters when set.
		ID string `validate:"omitempty,hetrixtools_id"`
		// Name filters monitors by partial or full monitor name.
		Name string
		// Target filters monitors by partial or full monitored target.
		Target string
		// Category filters monitors by partial or full category name.
		Category string
		// Type filters by the HetrixTools v3 monitor type enum: website, ping, service, smtp, or heartbeat.
		Type string `validate:"omitempty,oneof=website ping service smtp heartbeat"`
		// UptimeStatus filters by current uptime status. Accepted values are up and down.
		UptimeStatus string `validate:"omitempty,oneof=up down"`
		// MonitorStatus filters by monitor lifecycle status. Accepted values are active, paused, disabled, maint, and maint_dnd.
		MonitorStatus string `validate:"omitempty,oneof=active paused disabled maint maint_dnd"`
		// Order controls result sort direction. Accepted values are asc and desc.
		Order string `validate:"omitempty,oneof=asc desc"`
		// OrderBy selects the field used for sorting. Accepted values are name, created_at, last_check, last_status_change, uptime_status, and monitor_status.
		OrderBy string `validate:"omitempty,oneof=name created_at last_check last_status_change uptime_status monitor_status"`
	}

	// GetUptimeMonitorReportRequest filters uptime monitor report results.
	GetUptimeMonitorReportRequest struct {
		// Days is the number of recent days to display. HetrixTools accepts 1 through 30 and defaults to 7.
		Days int `validate:"omitempty,min=1,max=30"`
		// Month is the report month in YYYY-MM format. When set, HetrixTools ignores Days.
		Month string `validate:"omitempty,datetime=2006-01"`
		// Timezone is the report timezone offset, such as +02:00 or -5:30. HetrixTools defaults to +00:00.
		Timezone string
		// HourlyStats controls whether HetrixTools includes hourly stats for website uptime monitors.
		HourlyStats *bool
	}

	// ListUptimeMonitorDowntimesRequest filters uptime monitor downtime results.
	ListUptimeMonitorDowntimesRequest struct {
		// PaginationRequest contains page and per_page filters. Downtime lists accept per_page up to 200.
		PaginationRequest
		// StartBefore filters to downtime entries that started at or before this Unix timestamp.
		StartBefore int64 `validate:"omitempty,min=1"`
		// StartAfter filters to downtime entries that started at or after this Unix timestamp.
		StartAfter int64 `validate:"omitempty,min=1"`
	}

	// GetUptimeMonitorLocationFailLogRequest filters uptime monitor location fail logs.
	GetUptimeMonitorLocationFailLogRequest struct {
		// Timestamp is the Unix timestamp where HetrixTools starts scanning backward for log entries. Defaults to the current timestamp.
		Timestamp int64 `validate:"omitempty,min=1"`
		// Minutes is the number of minutes containing log entries to return. HetrixTools accepts 1 through 100 and defaults to 10.
		Minutes int `validate:"omitempty,min=1,max=100"`
	}

	// UptimeMonitor describes a HetrixTools uptime monitor.
	UptimeMonitor struct {
		// ID is the HetrixTools uptime monitor ID.
		ID string `json:"id"`
		// Type is the canonical monitor type: http, ping, smtp, or heartbeat.
		Type string `json:"-"`
		// Name is the human-readable monitor name.
		Name string `json:"name"`
		// Target is the URL, host, IP, or SMTP hostname checked by non-heartbeat monitors.
		Target string `json:"target"`
		// Port is the SMTP port. It is nil for non-SMTP monitors.
		Port *int64 `json:"port"`
		// HTTPMethod is the HTTP method for http monitors.
		HTTPMethod string `json:"http_method"`
		// MaxRedirects is the maximum redirects followed by http monitors.
		MaxRedirects int64 `json:"max_redirects"`
		// SMTPUser is the SMTP username returned by the API for smtp monitors.
		SMTPUser string `json:"smtp_user"`
		// Timeout is the check timeout in seconds.
		Timeout int64 `json:"timeout"`
		// Frequency is the check frequency in minutes.
		Frequency int64 `json:"frequency"`
		// FailsBeforeAlert is the number of failed checks required before alerting.
		FailsBeforeAlert int64 `json:"fails_before_alert"`
		// FailedLocations is the number of failed locations required before alerting.
		FailedLocations int64 `json:"failed_locations"`
		// ContactListID is the first contact list ID used for notifications.
		ContactListID string `json:"contact_list_id"`
		// ContactListIDs contains every contact list ID returned by the v3 API.
		ContactListIDs []string `json:"-"`
		// Category is the HetrixTools monitor category.
		Category string `json:"category"`
		// AlertAfter is the delay before sending an alert, such as 5m.
		AlertAfter string `json:"alert_after"`
		// RepeatTimes is the number of times to repeat alerts.
		RepeatTimes int64 `json:"repeat_times"`
		// RepeatEvery is the alert repeat interval, such as 60m.
		RepeatEvery string `json:"repeat_every"`
		// Public reports whether the monitor has a public report.
		Public *bool `json:"public"`
		// ShowTarget reports whether the monitor target is shown publicly.
		ShowTarget *bool `json:"show_target"`
		// VerSSLCert reports whether SSL certificate validation is enabled.
		VerSSLCert *bool `json:"verify_ssl_certificate"`
		// VerSSLHost reports whether SSL hostname validation is enabled.
		VerSSLHost *bool `json:"verify_ssl_host"`
		// Locations contains canonical monitoring location names enabled for the monitor.
		Locations []string `json:"-"`
		// Keyword is the expected response-body keyword for http monitors.
		Keyword string `json:"keyword"`
		// HTTPCodes contains accepted HTTP status codes for http monitors.
		HTTPCodes []int64 `json:"accepted_http_codes"`
		// Grace is the heartbeat grace period for heartbeat monitors.
		Grace int64 `json:"grace"`
		// InfoPublic reports whether heartbeat info details are public.
		InfoPublic *bool `json:"info_public"`
		// CPUPublic reports whether heartbeat CPU details are public.
		CPUPublic *bool `json:"cpu_public"`
		// RAMPublic reports whether heartbeat RAM details are public.
		RAMPublic *bool `json:"ram_public"`
		// DiskPublic reports whether heartbeat disk details are public.
		DiskPublic *bool `json:"disk_public"`
		// NetPublic reports whether heartbeat network details are public.
		NetPublic *bool `json:"net_public"`
		// SSLExpirationReminder is the number of days before SSL expiration to alert; zero disables it.
		SSLExpirationReminder int64 `json:"ssl_expiration_reminder"`
		// DomainExpirationReminder is the number of days before domain expiration to alert; zero disables it.
		DomainExpirationReminder int64 `json:"domain_expiration_reminder"`
		// NSChangeAlert reports whether nameserver-change alerts are enabled.
		NSChangeAlert *bool `json:"ns_change_alert"`
		// MonitorStatus is the remote lifecycle status, such as active or paused.
		MonitorStatus string `json:"monitor_status"`
		// UnknownFields contains response fields not understood by this client.
		UnknownFields []string `json:"-"`
		// PresentFields records raw API response field presence before normalization.
		PresentFields map[string]bool `json:"-"`
		// ServerID is the attached server-agent ID. It is nil for non-heartbeat monitors.
		ServerID *string `json:"server_id"`
	}

	// ListUptimeMonitorsResponse is returned by ListUptimeMonitors.
	ListUptimeMonitorsResponse struct {
		// UptimeMonitors contains the returned uptime monitors.
		UptimeMonitors []UptimeMonitor `json:"uptime_monitors"`
		// Meta contains pagination and result-count metadata.
		Meta Meta `json:"meta"`
		// MonitorsPresent reports whether the API response contained a recognized monitor list field.
		MonitorsPresent bool `json:"-"`
	}

	// ListUptimeMonitorDowntimesResponse is returned by ListUptimeMonitorDowntimes.
	ListUptimeMonitorDowntimesResponse struct {
		// Downtimes contains the returned downtime entries.
		Downtimes []UptimeMonitorDowntime `json:"downtimes"`
		// Meta contains pagination and result-count metadata.
		Meta Meta `json:"meta"`
	}

	// UptimeMonitorDowntime describes one uptime monitor downtime entry.
	UptimeMonitorDowntime struct {
		// ID is the unique downtime ID.
		ID string `json:"id"`
		// Start is the Unix timestamp when the downtime started.
		Start int64 `json:"start"`
		// End is the Unix timestamp when the downtime ended.
		End int64 `json:"end"`
		// Maintenance reports whether the downtime happened during a maintenance window.
		Maintenance bool `json:"maintenance"`
	}

	// UptimeMonitorLocationFailLogResponse is returned by GetUptimeMonitorLocationFailLog.
	UptimeMonitorLocationFailLogResponse struct {
		// Entries contains returned location fail-log entries.
		Entries []UptimeMonitorLocationFailLogEntry `json:"entries"`
		// Meta contains fail-log scan metadata.
		Meta UptimeMonitorLocationFailLogMeta `json:"meta"`
	}

	// UptimeMonitorLocationFailLogEntry describes one location fail-log entry.
	UptimeMonitorLocationFailLogEntry struct {
		// Timestamp is when the monitoring node encountered the error.
		Timestamp int64 `json:"timestamp"`
		// Location is the monitoring location and node that logged the error.
		Location string `json:"location"`
		// Data is the exact error encountered by the monitoring node.
		Data string `json:"data"`
	}

	// UptimeMonitorLocationFailLogMeta contains location fail-log scan metadata.
	UptimeMonitorLocationFailLogMeta struct {
		// Returned is the number of log entries returned.
		Returned int64 `json:"returned"`
		// StartTimestamp is the timestamp where scanning began.
		StartTimestamp int64 `json:"start_timestamp"`
		// EndTimestamp is the timestamp where scanning stopped.
		EndTimestamp int64 `json:"end_timestamp"`
		// NextPageTimestamp is the timestamp to request for the next page, or nil when there is no next page.
		NextPageTimestamp *int64 `json:"next_page_timestamp"`
	}

	// UptimeMonitorReportResponse is returned by GetUptimeMonitorReport.
	UptimeMonitorReportResponse struct {
		// Timezone is the timezone used in the report.
		Timezone string `json:"timezone"`
		// Data contains report datapoints keyed by date in YYYY-MM-DD format.
		Data map[string]UptimeReportData `json:"data"`
		// Summary contains aggregate uptime and response-time values.
		Summary UptimeReportSummary `json:"summary"`
		// History contains historical uptime data keyed by month in YYYY-MM format.
		History map[string]UptimeReportHistoryData `json:"history"`
	}

	// UptimeReportData contains one daily uptime report datapoint.
	UptimeReportData struct {
		// Uptime contains uptime percentages and downtime counts for this day.
		Uptime UptimeReportUptimeSummary `json:"uptime"`
		// ResponseTime contains average response times by monitoring location.
		ResponseTime UptimeReportLocationResponseTimes `json:"response_time"`
		// HourlyStats contains hourly response-time phase timings keyed by hour, when requested.
		HourlyStats map[string]UptimeReportHourlyStats `json:"hourly_stats"`
	}

	// UptimeReportHistoryData contains one monthly historical uptime datapoint.
	UptimeReportHistoryData struct {
		// Uptime contains uptime percentages for this historical month.
		Uptime UptimeReportUptimeSummary `json:"uptime"`
	}

	// UptimeReportSummary contains aggregate uptime report values.
	UptimeReportSummary struct {
		// Uptime contains aggregate uptime percentages and downtime counts.
		Uptime UptimeReportUptimeSummary `json:"uptime"`
		// ResponseTime contains aggregate response-time values by monitoring location.
		ResponseTime UptimeReportLocationResponseTimes `json:"response_time"`
	}

	// UptimeReportUptimeSummary contains aggregate uptime values.
	UptimeReportUptimeSummary struct {
		// Percentage is the overall uptime percentage.
		Percentage float64 `json:"percentage"`
		// PercentageInclMaint is the overall uptime percentage including maintenance.
		PercentageInclMaint float64 `json:"percentage_incl_maint"`
		// Downtimes is the total number of recorded downtimes.
		Downtimes int64 `json:"downtimes"`
		// DowntimesInclMaint is the total number of downtimes including maintenance.
		DowntimesInclMaint int64 `json:"downtimes_incl_maint"`
	}

	// UptimeReportLocationResponseTimes contains response-time values by HetrixTools monitoring location.
	UptimeReportLocationResponseTimes struct {
		// NewYork is the response time observed from New York.
		NewYork float64 `json:"new_york"`
		// SanFrancisco is the response time observed from San Francisco.
		SanFrancisco float64 `json:"san_francisco"`
		// Dallas is the response time observed from Dallas.
		Dallas float64 `json:"dallas"`
		// Amsterdam is the response time observed from Amsterdam.
		Amsterdam float64 `json:"amsterdam"`
		// London is the response time observed from London.
		London float64 `json:"london"`
		// Frankfurt is the response time observed from Frankfurt.
		Frankfurt float64 `json:"frankfurt"`
		// Singapore is the response time observed from Singapore.
		Singapore float64 `json:"singapore"`
		// Sydney is the response time observed from Sydney.
		Sydney float64 `json:"sydney"`
		// SaoPaulo is the response time observed from Sao Paulo.
		SaoPaulo float64 `json:"sao_paulo"`
		// Tokyo is the response time observed from Tokyo.
		Tokyo float64 `json:"tokyo"`
		// Mumbai is the response time observed from Mumbai.
		Mumbai float64 `json:"mumbai"`
		// Warsaw is the response time observed from Warsaw.
		Warsaw float64 `json:"warsaw"`
	}

	// UptimeReportHourlyStats contains hourly response-time phase timings by monitoring location.
	UptimeReportHourlyStats struct {
		// NewYork contains hourly timings observed from New York.
		NewYork UptimeReportHourlyTimings `json:"new_york"`
		// SanFrancisco contains hourly timings observed from San Francisco.
		SanFrancisco UptimeReportHourlyTimings `json:"san_francisco"`
		// Dallas contains hourly timings observed from Dallas.
		Dallas UptimeReportHourlyTimings `json:"dallas"`
		// Amsterdam contains hourly timings observed from Amsterdam.
		Amsterdam UptimeReportHourlyTimings `json:"amsterdam"`
		// London contains hourly timings observed from London.
		London UptimeReportHourlyTimings `json:"london"`
		// Frankfurt contains hourly timings observed from Frankfurt.
		Frankfurt UptimeReportHourlyTimings `json:"frankfurt"`
		// Singapore contains hourly timings observed from Singapore.
		Singapore UptimeReportHourlyTimings `json:"singapore"`
		// Sydney contains hourly timings observed from Sydney.
		Sydney UptimeReportHourlyTimings `json:"sydney"`
		// SaoPaulo contains hourly timings observed from Sao Paulo.
		SaoPaulo UptimeReportHourlyTimings `json:"sao_paulo"`
		// Tokyo contains hourly timings observed from Tokyo.
		Tokyo UptimeReportHourlyTimings `json:"tokyo"`
		// Mumbai contains hourly timings observed from Mumbai.
		Mumbai UptimeReportHourlyTimings `json:"mumbai"`
		// Warsaw contains hourly timings observed from Warsaw.
		Warsaw UptimeReportHourlyTimings `json:"warsaw"`
	}

	// UptimeReportHourlyTimings contains response-time phase timings in milliseconds.
	UptimeReportHourlyTimings struct {
		// DNSLookup is the DNS lookup duration.
		DNSLookup float64 `json:"dns_lookup"`
		// TCPConnect is the TCP connection duration.
		TCPConnect float64 `json:"tcp_connect"`
		// TLSConnect is the TLS handshake duration.
		TLSConnect float64 `json:"tls_connect"`
		// TTFB is the time to first byte.
		TTFB float64 `json:"ttfb"`
		// Download is the response download duration.
		Download float64 `json:"download"`
		// Total is the total observed response time.
		Total float64 `json:"total"`
	}
)

// MarshalJSON translates the canonical client model into the v2 add/update API
// shape documented by HetrixTools:
//
//   - https://docs.hetrixtools.com/api-add-website-ping-service-smtp-uptime-monitor/
//   - https://docs.hetrixtools.com/api-add-server-agent-uptime-monitor-heartbeat-uptime-monitor/
func (r UptimeMonitorRequest) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	type uptimeMonitorRequest UptimeMonitorRequest
	body, err := json.Marshal(uptimeMonitorRequest(r))
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if len(r.Locations) > 0 {
		locations, _ := uptimeLocationCodes(r.Locations)
		payload["Locations"] = locations
	}
	if r.Type != "" {
		typeID, _ := uptimeMonitorTypeID(r.Type)
		payload["Type"] = typeID
	}
	// Zero is meaningful for repeat counts and heartbeat grace. The default
	// encoding/json omitempty behavior would otherwise make preview claim a
	// convergent update that push cannot actually send.
	payload["RepeatTimes"] = r.RepeatTimes
	if r.Type == "heartbeat" {
		payload["Grace"] = r.Grace
	}
	// The edit API does not document how omitted parameters are treated, so
	// edits pin the expiration warnings explicitly; the documented value 0
	// means off. Create requests omit zeros to keep the server-side defaults.
	if r.MID != "" && r.Type == "http" {
		payload["SSLExpiryReminder"] = r.SSLExpirationReminder
		payload["DomainExpiryReminder"] = r.DomainExpirationReminder
	}
	// NSChangeAlert is documented as 1/0, unlike the true/false booleans.
	if r.NSChangeAlert != nil {
		nsChangeAlert := int64(0)
		if *r.NSChangeAlert {
			nsChangeAlert = 1
		}
		payload["NSChangeAlert"] = nsChangeAlert
	}
	if r.Keyword != "" {
		payload["Keyword"] = r.Keyword
	}
	if len(r.HTTPCodes) > 0 {
		payload["HTTPCodes"] = r.HTTPCodes
	}
	return json.Marshal(payload)
}

// Validate rejects combinations the documented HetrixTools v2 uptime APIs cannot
// represent.
func (r UptimeMonitorRequest) Validate() error {
	return validateRequest(r)
}

// UnmarshalJSON accepts both v3 snake_case fields and legacy v2 camel-case names.
func (m *UptimeMonitor) UnmarshalJSON(body []byte) error {
	type uptimeMonitor UptimeMonitor
	var aux struct {
		uptimeMonitor
		IDMonitorID           string          `json:"monitor_id"`
		IDMID                 string          `json:"MID"`
		IDCamel               string          `json:"MonitorID"`
		TypeRaw               json.RawMessage `json:"type"`
		TypeCamel             json.RawMessage `json:"Type"`
		AgentID               *string         `json:"agent_id"`
		PortCamel             *int64          `json:"Port"`
		HTTPMethodCamel       string          `json:"Method"`
		MaxRedirectsCamel     int64           `json:"MaxRedirects"`
		SMTPUserCamel         string          `json:"SMTPUser"`
		FrequencyV3           int64           `json:"check_frequency"`
		ContactLists          []string        `json:"contact_lists"`
		FailsBeforeAlertV3    int64           `json:"number_of_tries"`
		FailedLocationsV3     int64           `json:"triggering_locations"`
		AlertAfterMinutes     int64           `json:"alert_after_minutes"`
		RepeatTimesV3         int64           `json:"repeat_alert_times"`
		RepeatEveryV3         int64           `json:"repeat_alert_frequency"`
		PublicV3              *bool           `json:"public_report"`
		ShowTargetV3          *bool           `json:"public_target"`
		VerSSLHostV3          *bool           `json:"verify_ssl_hostname"`
		LocationsV3           map[string]any  `json:"locations"`
		ContactListCamel      string          `json:"ContactList"`
		FailsBeforeAlertCamel int64           `json:"FailsBeforeAlert"`
		FailedLocationsCamel  int64           `json:"FailedLocations"`
		AlertAfterCamel       string          `json:"AlertAfter"`
		RepeatTimesCamel      int64           `json:"RepeatTimes"`
		RepeatEveryCamel      string          `json:"RepeatEvery"`
		ShowTargetCamel       *bool           `json:"ShowTarget"`
		VerSSLCertCamel       *bool           `json:"VerSSLCert"`
		VerSSLHostCamel       *bool           `json:"VerSSLHost"`
		LocationsCamel        map[string]bool `json:"Locations"`
		InfoPublicCamel       *bool           `json:"INFOPub"`
		CPUPublicCamel        *bool           `json:"CPUPub"`
		RAMPublicCamel        *bool           `json:"RAMPub"`
		DiskPublicCamel       *bool           `json:"DISKPub"`
		NetPublicCamel        *bool           `json:"NETPub"`
		SSLExpirationWarn     bool            `json:"ssl_expiration_warn"`
		SSLExpirationDays     int64           `json:"ssl_expiration_warn_days"`
		DomainExpirationWarn  bool            `json:"domain_expiration_warn"`
		DomainExpirationDays  int64           `json:"domain_expiration_warn_days"`
		NSChangeWarn          *bool           `json:"nameservers_change_warn"`
	}
	if err := json.Unmarshal(body, &aux); err != nil {
		return err
	}
	*m = UptimeMonitor(aux.uptimeMonitor)
	if m.ID == "" {
		m.ID = firstNonEmpty(aux.IDMonitorID, aux.IDMID, aux.IDCamel)
	}
	if m.Type == "" {
		m.Type = uptimeMonitorTypeName(firstNonEmptyRawMessage(aux.TypeRaw, aux.TypeCamel))
	}
	if m.Frequency == 0 {
		m.Frequency = aux.FrequencyV3
	}
	if m.ServerID == nil {
		m.ServerID = aux.AgentID
	}
	if m.Type != "heartbeat" {
		m.ServerID = nil
	}
	if m.Port == nil {
		m.Port = aux.PortCamel
	}
	if m.Type != "smtp" {
		m.Port = nil
	}
	if m.HTTPMethod == "" {
		m.HTTPMethod = aux.HTTPMethodCamel
	}
	if m.MaxRedirects == 0 {
		m.MaxRedirects = aux.MaxRedirectsCamel
	}
	if m.SMTPUser == "" {
		m.SMTPUser = aux.SMTPUserCamel
	}
	if len(aux.ContactLists) > 0 {
		m.ContactListIDs = append([]string(nil), aux.ContactLists...)
	}
	if m.ContactListID == "" {
		if len(aux.ContactLists) > 0 {
			m.ContactListID = aux.ContactLists[0]
		} else {
			m.ContactListID = aux.ContactListCamel
		}
	}
	if m.FailsBeforeAlert == 0 {
		m.FailsBeforeAlert = firstNonZeroInt64(aux.FailsBeforeAlertV3, aux.FailsBeforeAlertCamel)
	}
	if m.FailedLocations == 0 {
		m.FailedLocations = firstNonZeroInt64(aux.FailedLocationsV3, aux.FailedLocationsCamel)
	}
	if m.AlertAfter == "" {
		m.AlertAfter = firstNonEmpty(durationMinutes(aux.AlertAfterMinutes), aux.AlertAfterCamel)
	}
	if m.RepeatTimes == 0 {
		m.RepeatTimes = firstNonZeroInt64(aux.RepeatTimesV3, aux.RepeatTimesCamel)
	}
	if m.RepeatEvery == "" {
		m.RepeatEvery = firstNonEmpty(durationMinutes(aux.RepeatEveryV3), aux.RepeatEveryCamel)
	}
	if m.Public == nil {
		m.Public = aux.PublicV3
	}
	if m.ShowTarget == nil {
		m.ShowTarget = aux.ShowTargetV3
	}
	if m.ShowTarget == nil {
		m.ShowTarget = aux.ShowTargetCamel
	}
	if m.VerSSLCert == nil {
		m.VerSSLCert = aux.VerSSLCertCamel
	}
	if m.VerSSLHost == nil {
		m.VerSSLHost = firstNonNilBool(aux.VerSSLHostV3, aux.VerSSLHostCamel)
	}
	if m.Locations == nil {
		if len(aux.LocationsV3) > 0 {
			m.Locations = make([]string, 0, len(aux.LocationsV3))
			for key := range aux.LocationsV3 {
				m.Locations = append(m.Locations, uptimeLocationName(key))
			}
			sort.Strings(m.Locations)
		} else {
			m.Locations = uptimeLocationNames(aux.LocationsCamel)
		}
	}
	if m.InfoPublic == nil {
		m.InfoPublic = aux.InfoPublicCamel
	}
	if m.CPUPublic == nil {
		m.CPUPublic = aux.CPUPublicCamel
	}
	if m.RAMPublic == nil {
		m.RAMPublic = aux.RAMPublicCamel
	}
	if m.DiskPublic == nil {
		m.DiskPublic = aux.DiskPublicCamel
	}
	if m.NetPublic == nil {
		m.NetPublic = aux.NetPublicCamel
	}
	if m.SSLExpirationReminder == 0 && aux.SSLExpirationWarn {
		m.SSLExpirationReminder = aux.SSLExpirationDays
	}
	if m.DomainExpirationReminder == 0 && aux.DomainExpirationWarn {
		m.DomainExpirationReminder = aux.DomainExpirationDays
	}
	if m.NSChangeAlert == nil {
		m.NSChangeAlert = aux.NSChangeWarn
	}
	m.UnknownFields, m.PresentFields = inspectUptimeMonitorFields(body)
	return nil
}

var knownUptimeMonitorFields = map[string]struct{}{
	"id": {}, "monitor_id": {}, "MID": {}, "MonitorID": {},
	"name": {}, "Name": {}, "type": {}, "Type": {}, "target": {}, "Target": {}, "resolve_address": {}, "resolve_address_info": {},
	"port": {}, "Port": {}, "keyword": {}, "Keyword": {}, "category": {}, "Category": {}, "timeout": {}, "Timeout": {}, "frequency": {}, "Frequency": {}, "check_frequency": {},
	"contact_list_id": {}, "contact_lists": {}, "created_at": {}, "last_check": {}, "last_status_change": {},
	"uptime_status": {}, "monitor_status": {}, "uptime": {}, "uptime_incl_maint": {}, "locations": {}, "Locations": {},
	"ssl_expiration_date": {}, "ssl_expiration_warn": {}, "ssl_expiration_warn_days": {},
	"domain_expiration_date": {}, "domain_expiration_warn": {}, "domain_expiration_warn_days": {},
	"nameservers": {}, "nameservers_change_warn": {}, "public_report": {}, "public_target": {},
	"max_redirects": {}, "MaxRedirects": {}, "http_method": {}, "Method": {}, "accepted_http_codes": {},
	"verify_ssl_certificate": {}, "verify_ssl_hostname": {}, "verify_ssl_host": {},
	"number_of_tries": {}, "fails_before_alert": {}, "FailsBeforeAlert": {},
	"triggering_locations": {}, "failed_locations": {}, "FailedLocations": {},
	"alert_after_minutes": {}, "alert_after": {}, "AlertAfter": {},
	"repeat_alert_times": {}, "repeat_times": {}, "RepeatTimes": {},
	"repeat_alert_frequency": {}, "repeat_every": {}, "RepeatEvery": {},
	"public": {}, "Public": {}, "show_target": {}, "ShowTarget": {}, "VerSSLCert": {}, "VerSSLHost": {},
	"smtp_user": {}, "SMTPUser": {}, "ContactList": {}, "HTTPCodes": {},
	"grace": {}, "Grace": {}, "info_public": {}, "cpu_public": {}, "ram_public": {}, "disk_public": {}, "net_public": {},
	"INFOPub": {}, "CPUPub": {}, "RAMPub": {}, "DISKPub": {}, "NETPub": {},
	"ssl_expiration_reminder": {}, "domain_expiration_reminder": {}, "ns_change_alert": {},
	"agent_id": {}, "server_id": {},
}

func inspectUptimeMonitorFields(body []byte) ([]string, map[string]bool) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return []string{"<unparseable>"}, nil
	}
	present := make(map[string]bool, len(fields))
	var unknown []string
	for field := range fields {
		present[field] = true
		if _, ok := knownUptimeMonitorFields[field]; !ok {
			unknown = append(unknown, field)
		}
	}
	sort.Strings(unknown)
	return unknown, present
}

// UnmarshalJSON accepts documented and legacy list envelope names.
func (r *ListUptimeMonitorsResponse) UnmarshalJSON(body []byte) error {
	type listUptimeMonitorsResponse ListUptimeMonitorsResponse
	var aux struct {
		listUptimeMonitorsResponse
		Monitors []UptimeMonitor `json:"monitors"`
		Data     []UptimeMonitor `json:"data"`
	}
	if err := json.Unmarshal(body, &aux); err != nil {
		return err
	}
	*r = ListUptimeMonitorsResponse(aux.listUptimeMonitorsResponse)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return err
	}
	var list json.RawMessage
	for _, field := range []string{"uptime_monitors", "monitors", "data"} {
		if raw, ok := fields[field]; ok {
			list = raw
			break
		}
	}
	if len(list) == 0 || bytes.Equal(bytes.TrimSpace(list), []byte("null")) {
		return fmt.Errorf("uptime-monitor response omitted a non-null monitor list")
	}
	_, hasCanonical := fields["uptime_monitors"]
	_, hasMonitors := fields["monitors"]
	_, hasData := fields["data"]
	r.MonitorsPresent = hasCanonical || hasMonitors || hasData
	if len(r.UptimeMonitors) == 0 {
		r.UptimeMonitors = firstNonEmptySlice(aux.Monitors, aux.Data)
	}
	return nil
}

func uptimeLocationNames(locations map[string]bool) []string {
	if locations == nil {
		return nil
	}
	normalized := make([]string, 0, len(locations))
	for key, value := range locations {
		if value {
			normalized = append(normalized, uptimeLocationName(key))
		}
	}
	sort.Strings(normalized)
	return normalized
}

func uptimeLocationCodes(locations []string) (map[string]bool, error) {
	if locations == nil {
		return nil, nil
	}
	normalized := map[string]bool{}
	for _, location := range locations {
		code, ok := uptimeLocationCode(location)
		if !ok {
			return nil, fmt.Errorf("unknown uptime monitor location %q", location)
		}
		normalized[code] = true
	}
	return normalized, nil
}

func uptimeLocationName(location string) string {
	switch location {
	case "nyc":
		return "new_york"
	case "sfo":
		return "san_francisco"
	case "dal":
		return "dallas"
	case "ams":
		return "amsterdam"
	case "lon":
		return "london"
	case "fra":
		return "frankfurt"
	case "sgp":
		return "singapore"
	case "syd":
		return "sydney"
	case "sao":
		return "sao_paulo"
	case "tok":
		return "tokyo"
	case "mba":
		return "mumbai"
	case "waw":
		return "warsaw"
	default:
		return location
	}
}

func uptimeLocationCode(location string) (string, bool) {
	switch location {
	case "new_york":
		return "nyc", true
	case "san_francisco":
		return "sfo", true
	case "dallas":
		return "dal", true
	case "amsterdam":
		return "ams", true
	case "london":
		return "lon", true
	case "frankfurt":
		return "fra", true
	case "singapore":
		return "sgp", true
	case "sydney":
		return "syd", true
	case "sao_paulo":
		return "sao", true
	case "tokyo":
		return "tok", true
	case "mumbai":
		return "mba", true
	case "warsaw":
		return "waw", true
	default:
		return "", false
	}
}

func durationMinutes(minutes int64) string {
	if minutes == 0 {
		return ""
	}
	return strconv.FormatInt(minutes, 10) + "m"
}

func uptimeMonitorTypeName(raw json.RawMessage) string {
	var id int64
	if err := json.Unmarshal(raw, &id); err == nil {
		switch id {
		case 1:
			return "http"
		case 2:
			return "ping"
		case 3:
			return "smtp"
		case 9:
			return "heartbeat"
		default:
			return ""
		}
	}
	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		return ""
	}
	switch name {
	case "website", "http":
		return "http"
	case "ping", "service":
		return "ping"
	case "smtp":
		return "smtp"
	case "server", "server_agent", "heartbeat":
		return "heartbeat"
	default:
		return name
	}
}

func uptimeMonitorTypeID(name string) (int64, error) {
	switch name {
	case "http":
		return 1, nil
	case "ping":
		return 2, nil
	case "smtp":
		return 3, nil
	case "heartbeat":
		return 9, nil
	default:
		return 0, fmt.Errorf("unknown uptime monitor type %q", name)
	}
}

func (r ListUptimeMonitorsRequest) query() map[string]string {
	values := map[string]string{}
	r.PaginationRequest.appendQuery(values)
	setString(values, "id", r.ID)
	setString(values, "name", r.Name)
	setString(values, "target", r.Target)
	setString(values, "category", r.Category)
	setString(values, "type", r.Type)
	setString(values, "uptime_status", r.UptimeStatus)
	setString(values, "monitor_status", r.MonitorStatus)
	setString(values, "order", r.Order)
	setString(values, "order_by", r.OrderBy)
	return values
}

func (r GetUptimeMonitorReportRequest) query() map[string]string {
	values := map[string]string{}
	setInt(values, "days", r.Days)
	setString(values, "month", r.Month)
	setString(values, "timezone", r.Timezone)
	setBool(values, "hourly_stats", r.HourlyStats)
	return values
}

func (r ListUptimeMonitorDowntimesRequest) query() map[string]string {
	values := map[string]string{}
	r.PaginationRequest.appendQuery(values)
	setInt64(values, "start_before", r.StartBefore)
	setInt64(values, "start_after", r.StartAfter)
	return values
}

func (r GetUptimeMonitorLocationFailLogRequest) query() map[string]string {
	values := map[string]string{}
	setInt64(values, "timestamp", r.Timestamp)
	setInt(values, "minutes", r.Minutes)
	return values
}

// CreateUptimeMonitor creates a HetrixTools uptime monitor using the documented
// v2 uptime add endpoint:
//
//   - https://docs.hetrixtools.com/api-add-website-ping-service-smtp-uptime-monitor/
//   - https://docs.hetrixtools.com/api-add-server-agent-uptime-monitor-heartbeat-uptime-monitor/
func (c *Client) CreateUptimeMonitor(ctx context.Context, request UptimeMonitorRequest) (*ActionResponse, error) {
	request.MID = ""
	body, err := c.doV2JSON(ctx, http.MethodPost, "/uptime/add/", request)
	if err != nil {
		return nil, err
	}
	c.clearMonitorCaches()
	return decodeActionResponse(body)
}

// UpdateUptimeMonitor updates a HetrixTools uptime monitor using the v2 uptime
// edit endpoint with MID set:
//
//   - https://gist.github.com/hetrixtools/b6fed467837874491f2a046fac957293
func (c *Client) UpdateUptimeMonitor(ctx context.Context, request UptimeMonitorRequest) (*ActionResponse, error) {
	if request.MID == "" {
		return nil, fmt.Errorf("update uptime monitor: MID is required")
	}
	body, err := c.doV2JSON(ctx, http.MethodPost, "/uptime/edit/", request)
	if err != nil {
		return nil, err
	}
	c.clearMonitorCaches()
	return decodeActionResponse(body)
}

// UpsertUptimeMonitor updates an uptime monitor when MID is set, otherwise it
// creates one. It calls CreateUptimeMonitor or UpdateUptimeMonitor; see those
// methods for source API references.
func (c *Client) UpsertUptimeMonitor(ctx context.Context, request UptimeMonitorRequest) (*ActionResponse, error) {
	if request.MID == "" {
		return c.CreateUptimeMonitor(ctx, request)
	}
	return c.UpdateUptimeMonitor(ctx, request)
}

// DeleteUptimeMonitor deletes a HetrixTools uptime monitor by monitor ID using
// the documented v2 uptime delete endpoint:
//
//   - https://docs.hetrixtools.com/api-delete-uptime-monitor/
func (c *Client) DeleteUptimeMonitor(ctx context.Context, monitorID string) error {
	if monitorID == "" {
		return fmt.Errorf("delete uptime monitor: monitor ID is required")
	}
	body, err := c.doV2JSON(ctx, http.MethodPost, "/uptime/delete/", map[string]string{"MID": monitorID})
	if err == nil {
		_, err = decodeActionResponse(body)
	}
	if err == nil {
		c.clearMonitorCaches()
	}
	return err
}

// ListUptimeMonitors returns uptime monitors matching query filters.
// Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1uptime-monitors/get
func (c *Client) ListUptimeMonitors(ctx context.Context, request ListUptimeMonitorsRequest) (*ListUptimeMonitorsResponse, error) {
	if err := validateQuery(request); err != nil {
		return nil, err
	}
	var response ListUptimeMonitorsResponse
	if err := c.getJSON(ctx, "/uptime-monitors", request.query(), &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// GetUptimeMonitor finds an uptime monitor by monitor ID using
// ListUptimeMonitors; see ListUptimeMonitors for the source API reference.
func (c *Client) GetUptimeMonitor(ctx context.Context, monitorID string) (*UptimeMonitor, error) {
	if monitorID == "" {
		return nil, nil
	}
	monitors, err := c.cachedUptimeMonitors(ctx)
	if err != nil {
		return nil, err
	}
	for _, monitor := range monitors {
		if monitor.ID == monitorID {
			monitor := monitor
			return &monitor, nil
		}
	}
	return nil, nil
}

func (c *Client) cachedUptimeMonitors(ctx context.Context) ([]UptimeMonitor, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	if c.uptimeMonitors != nil {
		return c.uptimeMonitors, nil
	}

	var monitors []UptimeMonitor
	for page := 1; ; page++ {
		response, err := c.ListUptimeMonitors(ctx, ListUptimeMonitorsRequest{PaginationRequest: PaginationRequest{Page: page, PerPage: 100}})
		if err != nil {
			return nil, err
		}
		monitors = append(monitors, response.UptimeMonitors...)
		if response.Meta.Pagination.Next == nil || page >= response.Meta.Pagination.Last {
			c.uptimeMonitors = monitors
			return c.uptimeMonitors, nil
		}
	}
}

// GetUptimeMonitorReport returns a report for an uptime monitor.
// Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1uptime-monitors~1{monitor_id}~1report/get
func (c *Client) GetUptimeMonitorReport(ctx context.Context, monitorID string, request GetUptimeMonitorReportRequest) (*UptimeMonitorReportResponse, error) {
	if err := validateQuery(request); err != nil {
		return nil, err
	}
	var response UptimeMonitorReportResponse
	if err := c.getJSON(ctx, "/uptime-monitors/"+monitorID+"/report", request.query(), &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// ListUptimeMonitorDowntimes returns downtime entries for an uptime monitor.
// Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1uptime-monitors~1{monitor_id}~1downtimes/get
func (c *Client) ListUptimeMonitorDowntimes(ctx context.Context, monitorID string, request ListUptimeMonitorDowntimesRequest) (*ListUptimeMonitorDowntimesResponse, error) {
	if err := validateQuery(request); err != nil {
		return nil, err
	}
	var response ListUptimeMonitorDowntimesResponse
	if err := c.getJSON(ctx, "/uptime-monitors/"+monitorID+"/downtimes", request.query(), &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// GetUptimeMonitorLocationFailLog returns location failure logs for an uptime
// monitor. Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1uptime-monitors~1{monitor_id}~1location-fail-log/get
func (c *Client) GetUptimeMonitorLocationFailLog(ctx context.Context, monitorID string, request GetUptimeMonitorLocationFailLogRequest) (*UptimeMonitorLocationFailLogResponse, error) {
	if err := validateQuery(request); err != nil {
		return nil, err
	}
	var response UptimeMonitorLocationFailLogResponse
	if err := c.getJSON(ctx, "/uptime-monitors/"+monitorID+"/location-fail-log", request.query(), &response); err != nil {
		return nil, err
	}
	return &response, nil
}
