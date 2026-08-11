package hetrixtools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

type (
	// BlacklistCheckResult is the result of a one-off blacklist check.
	BlacklistCheckResult struct {
		// Status is the API status string.
		Status string `json:"status"`
		// ErrorMessage contains the API error message when a check fails.
		ErrorMessage string `json:"error_message"`
		// APICallsLeft is the remaining API call allowance reported by HetrixTools.
		APICallsLeft int64 `json:"api_calls_left"`
		// BlacklistCheckCreditsLeft is the remaining blacklist-check credit count.
		BlacklistCheckCreditsLeft int64 `json:"blacklist_check_credits_left"`
		// BlacklistedCount is the number of blacklists currently listing the target.
		BlacklistedCount int64 `json:"blacklisted_count"`
		// BlacklistedOn lists the individual blacklists that include the target.
		BlacklistedOn []BlacklistListing `json:"blacklisted_on"`
		// Links contains HetrixTools report URLs for the check.
		Links BlacklistCheckLinks `json:"links"`
	}

	// BlacklistListing identifies one blacklist where a target is listed.
	BlacklistListing struct {
		// RBL is the blacklist provider name.
		RBL string `json:"rbl"`
		// Delist is the provider's delisting URL when returned by the API.
		Delist string `json:"delist"`
	}

	// BlacklistCheckLinks contains report URLs returned by a blacklist check.
	BlacklistCheckLinks struct {
		// ReportLink is the hosted HetrixTools report URL.
		ReportLink string `json:"report_link"`
		// WhitelabelReportLink is the whitelabel report URL when available.
		WhitelabelReportLink string `json:"whitelabel_report_link"`
		// APIReportLink is the API report URL.
		APIReportLink string `json:"api_report_link"`
		// APIBlacklistCheckLink is the API URL for repeating the check.
		APIBlacklistCheckLink string `json:"api_blacklist_check_link"`
	}

	// BlacklistMonitorRequest describes a blacklist monitor create or update request.
	//
	// The client sends this request to the documented HetrixTools v2 blacklist
	// monitor endpoints:
	//
	//   - Add: https://docs.hetrixtools.com/api-add-blacklist-monitor/
	//   - Edit: https://docs.hetrixtools.com/api-edit-blacklist-monitor/
	BlacklistMonitorRequest struct {
		// Target is an IP address, IP range, CIDR block, or domain name.
		Target string `validate:"required,blacklist_monitor_target"`
		// Label is an optional human-readable monitor label.
		Label string `validate:"omitempty,blacklist_monitor_name"`
		// Contact is an optional HetrixTools contact list ID.
		Contact string `validate:"omitempty,hetrixtools_id"`
	}

	// ListBlacklistMonitorsRequest filters blacklist monitor list results.
	ListBlacklistMonitorsRequest struct {
		// PaginationRequest contains page and per_page filters. Blacklist monitors accept per_page up to 1024.
		PaginationRequest
		// Name filters monitors by partial or full monitor name. HetrixTools accepts letters, numbers, spaces, dots, and hyphens.
		Name string `validate:"omitempty,blacklist_monitor_name"`
		// ExactName makes Name an exact-match filter. When true, HetrixTools ignores all other filters.
		ExactName *bool
		// Target filters monitors by partial or full monitored target. HetrixTools accepts letters, numbers, dots, and hyphens.
		Target string `validate:"omitempty,blacklist_monitor_target"`
		// ExactTarget makes Target an exact-match filter. When true, HetrixTools ignores all other filters.
		ExactTarget *bool
		// CIDR treats Target as an IPv4 CIDR range using this prefix length. HetrixTools accepts values from 21 through 32.
		CIDR int `validate:"omitempty,min=21,max=32"`
		// Type filters by blacklist monitor type. HetrixTools v3 supports ipv4 and domain.
		Type string `validate:"omitempty,oneof=ipv4 domain"`
		// Listed filters by whether the monitor is currently listed on any blacklist.
		Listed *bool
		// Order controls result sort direction. Accepted values are asc and desc.
		Order string `validate:"omitempty,oneof=asc desc"`
		// OrderBy selects the field used for sorting. Accepted values are name, target, listed, created_at, and last_check.
		OrderBy string `validate:"omitempty,oneof=name target listed created_at last_check"`
	}

	// GetBlacklistMonitorReportRequest filters blacklist monitor report results.
	GetBlacklistMonitorReportRequest struct {
		// Date is the report date in YYYY-MM-DD format. Leave empty to use HetrixTools' default report date.
		Date string `validate:"omitempty,datetime=2006-01-02"`
	}

	// BlacklistMonitorReportResponse is returned by GetBlacklistMonitorReport.
	BlacklistMonitorReportResponse struct {
		// ID is the unique blacklist monitor ID.
		ID string `json:"id"`
		// Name is the blacklist monitor name.
		Name string `json:"name"`
		// Type is the blacklist monitor type, either ipv4 or domain.
		Type string `json:"type"`
		// Target is the monitored target.
		Target string `json:"target"`
		// ReportID is the unique blacklist report ID.
		ReportID string `json:"report_id"`
		// Listed contains blacklists that listed the target on the requested date.
		Listed []BlacklistListing `json:"listed"`
	}

	// BlacklistMonitor describes a HetrixTools blacklist monitor.
	BlacklistMonitor struct {
		// ID is the HetrixTools blacklist monitor ID.
		ID string `json:"id"`
		// Target is the monitored IP address, CIDR block, range, or domain.
		Target string `json:"target"`
		// Label is the monitor label.
		Label string `json:"label"`
		// Name is an alternate monitor name returned by some API views.
		Name string `json:"name"`
		// Contact is the contact list ID used for notifications.
		Contact string `json:"contact"`
	}

	// ListBlacklistMonitorsResponse is returned by ListBlacklistMonitors.
	ListBlacklistMonitorsResponse struct {
		// BlacklistMonitors contains the returned blacklist monitors.
		BlacklistMonitors []BlacklistMonitor `json:"blacklist_monitors"`
		// Meta contains pagination and result-count metadata.
		Meta Meta `json:"meta"`
	}
)

// UnmarshalJSON accepts alternate field names returned by HetrixTools views.
func (m *BlacklistMonitor) UnmarshalJSON(body []byte) error {
	type blacklistMonitor BlacklistMonitor
	var aux struct {
		blacklistMonitor
		TargetIP     string   `json:"ip"`
		TargetDomain string   `json:"domain"`
		TargetHost   string   `json:"host"`
		Contacts     []string `json:"contact_lists"`
	}
	if err := json.Unmarshal(body, &aux); err != nil {
		return err
	}
	*m = BlacklistMonitor(aux.blacklistMonitor)
	if m.Target == "" {
		m.Target = firstNonEmpty(aux.TargetIP, aux.TargetDomain, aux.TargetHost)
	}
	if m.Contact == "" && len(aux.Contacts) > 0 {
		m.Contact = aux.Contacts[0]
	}
	return nil
}

// UnmarshalJSON accepts documented and legacy list envelope names.
func (r *ListBlacklistMonitorsResponse) UnmarshalJSON(body []byte) error {
	type listBlacklistMonitorsResponse ListBlacklistMonitorsResponse
	var aux struct {
		listBlacklistMonitorsResponse
		Monitors []BlacklistMonitor `json:"monitors"`
		Data     []BlacklistMonitor `json:"data"`
	}
	if err := json.Unmarshal(body, &aux); err != nil {
		return err
	}
	*r = ListBlacklistMonitorsResponse(aux.listBlacklistMonitorsResponse)
	if len(r.BlacklistMonitors) == 0 {
		r.BlacklistMonitors = firstNonEmptySlice(aux.Monitors, aux.Data)
	}
	return nil
}

func (r BlacklistMonitorRequest) form() url.Values {
	values := url.Values{"target": {r.Target}}
	if r.Label != "" {
		values.Set("label", r.Label)
	}
	if r.Contact != "" {
		values.Set("contact", r.Contact)
	}
	return values
}

func (r ListBlacklistMonitorsRequest) query() map[string]string {
	values := map[string]string{}
	r.PaginationRequest.appendQuery(values)
	setString(values, "name", r.Name)
	setBool(values, "exact_name", r.ExactName)
	setString(values, "target", r.Target)
	setBool(values, "exact_target", r.ExactTarget)
	setInt(values, "cidr", r.CIDR)
	setString(values, "type", r.Type)
	setBool(values, "listed", r.Listed)
	setString(values, "order", r.Order)
	setString(values, "order_by", r.OrderBy)
	return values
}

func (r GetBlacklistMonitorReportRequest) query() map[string]string {
	values := map[string]string{}
	setString(values, "date", r.Date)
	return values
}

// CreateBlacklistMonitor creates a HetrixTools blacklist monitor using the
// documented v2 blacklist add endpoint:
//
//   - https://docs.hetrixtools.com/api-add-blacklist-monitor/
func (c *Client) CreateBlacklistMonitor(ctx context.Context, request BlacklistMonitorRequest) (*ActionResponse, error) {
	if err := validateRequest(request); err != nil {
		return nil, err
	}
	body, err := c.doV2Form(ctx, "/blacklist/add/", request.form())
	if err != nil {
		return nil, err
	}
	c.clearMonitorCaches()
	return decodeActionResponse(body)
}

// UpdateBlacklistMonitor updates a HetrixTools blacklist monitor using the
// documented v2 blacklist edit endpoint:
//
//   - https://docs.hetrixtools.com/api-edit-blacklist-monitor/
func (c *Client) UpdateBlacklistMonitor(ctx context.Context, request BlacklistMonitorRequest) (*ActionResponse, error) {
	if err := validateRequest(request); err != nil {
		return nil, err
	}
	body, err := c.doV2Form(ctx, "/blacklist/edit/", request.form())
	if err != nil {
		return nil, err
	}
	c.clearMonitorCaches()
	return decodeActionResponse(body)
}

// UpsertBlacklistMonitor updates an existing blacklist monitor by target or
// creates it when absent. It calls GetBlacklistMonitor, CreateBlacklistMonitor,
// and UpdateBlacklistMonitor; see those methods for source API references.
func (c *Client) UpsertBlacklistMonitor(ctx context.Context, request BlacklistMonitorRequest) (*ActionResponse, error) {
	if err := validateRequest(request); err != nil {
		return nil, err
	}
	existing, err := c.GetBlacklistMonitor(ctx, request.Target)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return c.CreateBlacklistMonitor(ctx, request)
	}
	return c.UpdateBlacklistMonitor(ctx, request)
}

// DeleteBlacklistMonitor deletes a HetrixTools blacklist monitor by target using
// the documented v2 blacklist delete endpoint:
//
//   - https://docs.hetrixtools.com/api-delete-blacklist-monitor/
func (c *Client) DeleteBlacklistMonitor(ctx context.Context, target string) error {
	_, err := c.doV2Form(ctx, "/blacklist/delete/", url.Values{"target": {target}})
	if err == nil {
		c.clearMonitorCaches()
	}
	return err
}

// ListBlacklistMonitors returns blacklist monitors matching query filters.
// Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1blacklist-monitors/get
func (c *Client) ListBlacklistMonitors(ctx context.Context, request ListBlacklistMonitorsRequest) (*ListBlacklistMonitorsResponse, error) {
	if err := validateQuery(request); err != nil {
		return nil, err
	}
	var response ListBlacklistMonitorsResponse
	if err := c.getJSON(ctx, "/blacklist-monitors", request.query(), &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// GetBlacklistMonitor finds a blacklist monitor by exact target using
// ListBlacklistMonitors; see ListBlacklistMonitors for the source API reference.
func (c *Client) GetBlacklistMonitor(ctx context.Context, target string) (*BlacklistMonitor, error) {
	monitors, err := c.cachedBlacklistMonitors(ctx)
	if err != nil {
		return nil, err
	}
	for _, monitor := range monitors {
		if monitor.Target == target {
			monitor := monitor
			return &monitor, nil
		}
	}
	return nil, nil
}

func (c *Client) cachedBlacklistMonitors(ctx context.Context) ([]BlacklistMonitor, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	if c.blMonitors != nil {
		return c.blMonitors, nil
	}

	var monitors []BlacklistMonitor
	for page := 1; ; page++ {
		response, err := c.ListBlacklistMonitors(ctx, ListBlacklistMonitorsRequest{PaginationRequest: PaginationRequest{Page: page, PerPage: 100}})
		if err != nil {
			return nil, err
		}
		monitors = append(monitors, response.BlacklistMonitors...)
		if response.Meta.Pagination.Next == nil || page >= response.Meta.Pagination.Last {
			c.blMonitors = monitors
			return c.blMonitors, nil
		}
	}
}

// GetBlacklistMonitorReport returns the report for a blacklist monitor
// identifier. Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1blacklist-monitors~1{identifier}~1report/get
func (c *Client) GetBlacklistMonitorReport(ctx context.Context, identifier string, request GetBlacklistMonitorReportRequest) (*BlacklistMonitorReportResponse, error) {
	if err := validateQuery(request); err != nil {
		return nil, err
	}
	var response BlacklistMonitorReportResponse
	if err := c.getJSON(ctx, "/blacklist-monitors/"+identifier+"/report", request.query(), &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// CheckBlacklistIPv4 runs a one-off IPv4 blacklist check using the documented
// v2 blacklist check endpoint. Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/blacklist-check-api/
func (c *Client) CheckBlacklistIPv4(ctx context.Context, ipAddress string) (*BlacklistCheckResult, error) {
	return c.checkBlacklist(ctx, "ipv4", ipAddress)
}

// CheckBlacklistDomain runs a one-off domain blacklist check using the
// documented v2 blacklist check endpoint. Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/blacklist-check-api/
func (c *Client) CheckBlacklistDomain(ctx context.Context, domain string) (*BlacklistCheckResult, error) {
	return c.checkBlacklist(ctx, "domain", domain)
}

func (c *Client) checkBlacklist(ctx context.Context, kind string, target string) (*BlacklistCheckResult, error) {
	body, err := c.doV2JSON(ctx, http.MethodGet, "/blacklist-check/"+kind+"/"+url.PathEscape(target)+"/", nil)
	if err != nil {
		return nil, err
	}
	var result BlacklistCheckResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
