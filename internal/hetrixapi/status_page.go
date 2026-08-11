// Modified by the hetrixtools-sync project from the Apache-2.0 upstream source.
package hetrixtools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type (
	// StatusPage describes a HetrixTools status page.
	StatusPage struct {
		// ID is the status page ID.
		ID string `json:"id"`
		// Name is the status page name.
		Name string `json:"name"`
		// Type is the status page type returned by HetrixTools.
		Type string `json:"type"`
		// Monitors contains monitor IDs attached to the status page.
		Monitors []string `json:"monitors"`
		// MonitorsPresent reports whether the API response included membership.
		MonitorsPresent bool `json:"-"`
	}

	// ListStatusPagesRequest filters status page list results.
	ListStatusPagesRequest struct {
		// PaginationRequest contains page and per_page filters. Status pages accept per_page up to 100.
		PaginationRequest
		// Name filters status pages by status page name.
		Name string
	}

	// ListStatusPagesResponse is returned by ListStatusPages.
	ListStatusPagesResponse struct {
		// StatusPages contains the returned status pages.
		StatusPages []StatusPage `json:"status_pages"`
		// Meta contains pagination and result-count metadata.
		Meta Meta `json:"meta"`
	}
)

// UnmarshalJSON tracks whether status-page membership was present so callers
// cannot mistake an incomplete response for an empty page.
func (p *StatusPage) UnmarshalJSON(body []byte) error {
	type statusPage StatusPage
	var value statusPage
	if err := json.Unmarshal(body, &value); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return err
	}
	*p = StatusPage(value)
	raw, ok := fields["monitors"]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("status-page response omitted non-null monitor membership")
	}
	p.MonitorsPresent = true
	return nil
}

func (r ListStatusPagesRequest) query() map[string]string {
	values := map[string]string{}
	r.PaginationRequest.appendQuery(values)
	setString(values, "name", r.Name)
	return values
}

// ListStatusPages returns status pages matching query filters.
// Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1status-pages/get
func (c *Client) ListStatusPages(ctx context.Context, request ListStatusPagesRequest) (*ListStatusPagesResponse, error) {
	if err := validateQuery(request); err != nil {
		return nil, err
	}
	var response ListStatusPagesResponse
	if err := c.getJSON(ctx, "/status-pages", request.query(), &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// GetStatusPage finds a status page by ID using ListStatusPages; see
// ListStatusPages for the source API reference.
func (c *Client) GetStatusPage(ctx context.Context, id string) (*StatusPage, error) {
	for page := 1; ; page++ {
		response, err := c.ListStatusPages(ctx, ListStatusPagesRequest{PaginationRequest: PaginationRequest{Page: page, PerPage: 100}})
		if err != nil {
			return nil, err
		}
		for _, statusPage := range response.StatusPages {
			if statusPage.ID == id {
				return &statusPage, nil
			}
		}
		if response.Meta.Pagination.Next == nil || page >= response.Meta.Pagination.Last {
			return nil, nil
		}
	}
}

// AddStatusPageMonitors adds uptime monitors to a status page.
// Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1status-pages~1{status_page_id}~1monitors/post
func (c *Client) AddStatusPageMonitors(ctx context.Context, statusPageID string, monitorIDs []string) error {
	body, err := c.doV3(ctx, http.MethodPost, "/status-pages/"+statusPageID+"/monitors", nil, monitorIDs)
	if err != nil {
		return err
	}
	return validateStatusPageMutationResponse(body)
}

// RemoveStatusPageMonitors removes uptime monitors from a status page.
// Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1status-pages~1{status_page_id}~1monitors/delete
func (c *Client) RemoveStatusPageMonitors(ctx context.Context, statusPageID string, monitorIDs []string) error {
	body, err := c.doV3(ctx, http.MethodDelete, "/status-pages/"+statusPageID+"/monitors", nil, monitorIDs)
	if err != nil {
		return err
	}
	return validateStatusPageMutationResponse(body)
}

func validateStatusPageMutationResponse(body []byte) error {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("{}")) {
		return nil
	}
	var envelope struct {
		Status       string `json:"status"`
		Message      string `json:"message"`
		ErrorMessage string `json:"error_message"`
	}
	if err := json.Unmarshal(trimmed, &envelope); err != nil {
		return fmt.Errorf("unexpected status-page mutation response: %w", err)
	}
	if strings.EqualFold(envelope.Status, "success") && envelope.ErrorMessage == "" {
		return nil
	}
	message := envelope.ErrorMessage
	if message == "" {
		message = envelope.Message
	}
	if message == "" {
		message = fmt.Sprintf("unexpected status %q", envelope.Status)
	}
	return fmt.Errorf("status-page mutation failed: %s", message)
}
