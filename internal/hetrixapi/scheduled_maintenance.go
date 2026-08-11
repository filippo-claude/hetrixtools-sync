package hetrixtools

import "context"

type (
	// ScheduledMaintenanceRequest describes a scheduled maintenance create request.
	ScheduledMaintenanceRequest struct {
		// MonitorID is the uptime monitor ID that will be under maintenance.
		MonitorID string `json:"monitor_id"`
		// Start is the maintenance start timestamp in HetrixTools format.
		Start string `json:"start"`
		// End is the maintenance end timestamp in HetrixTools format.
		End string `json:"end"`
		// Timezone is the timezone used by Start and End.
		Timezone string `json:"timezone"`
		// WithNotifications controls whether HetrixTools sends maintenance notifications.
		WithNotifications bool `json:"with_notifications"`
		// RecurringTime is the recurrence interval value when recurrence is enabled.
		RecurringTime int64 `json:"recurring_time"`
		// RecurringTimeType is the recurrence unit used with RecurringTime.
		RecurringTimeType string `json:"recurring_time_type"`
	}

	// ScheduledMaintenance describes a HetrixTools scheduled maintenance window.
	ScheduledMaintenance struct {
		// ID is the scheduled maintenance ID.
		ID string `json:"id"`
		// MonitorID is the uptime monitor ID covered by the maintenance window.
		MonitorID string `json:"monitor_id"`
		// Start is the maintenance start timestamp returned by HetrixTools.
		Start string `json:"start"`
		// End is the maintenance end timestamp returned by HetrixTools.
		End string `json:"end"`
		// Timezone is the timezone used by Start and End.
		Timezone string `json:"timezone"`
		// WithNotifications reports whether notifications are enabled for the window.
		WithNotifications bool `json:"with_notifications"`
		// Recurring reports whether the maintenance window recurs.
		Recurring bool `json:"recurring"`
		// RecurringTime is the recurrence interval value.
		RecurringTime int64 `json:"recurring_time"`
		// RecurringTimeType is the recurrence unit used with RecurringTime.
		RecurringTimeType string `json:"recurring_time_type"`
	}

	// ListScheduledMaintenancesRequest filters scheduled maintenance list results.
	ListScheduledMaintenancesRequest struct {
		// PaginationRequest contains page and per_page filters. Scheduled maintenance lists accept per_page up to 200.
		PaginationRequest
		// MonitorID filters scheduled maintenance windows by uptime monitor ID.
		MonitorID string `validate:"omitempty,hetrixtools_id"`
	}

	// ListScheduledMaintenancesResponse is returned by ListScheduledMaintenances.
	ListScheduledMaintenancesResponse struct {
		// ScheduledMaintenances contains the returned maintenance windows.
		ScheduledMaintenances []ScheduledMaintenance `json:"scheduled_maintenances"`
		// Meta contains pagination and result-count metadata.
		Meta Meta `json:"meta"`
	}
)

func (r ListScheduledMaintenancesRequest) query() map[string]string {
	values := map[string]string{}
	r.PaginationRequest.appendQuery(values)
	setString(values, "monitor_id", r.MonitorID)
	return values
}

// CreateScheduledMaintenance creates a scheduled maintenance window.
// Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1schedule-maintenance/post
func (c *Client) CreateScheduledMaintenance(ctx context.Context, request ScheduledMaintenanceRequest) (*ScheduledMaintenance, error) {
	var created ScheduledMaintenance
	if err := c.postJSON(ctx, "/schedule-maintenance", request, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// DeleteScheduledMaintenance deletes a scheduled maintenance window by ID.
// Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1schedule-maintenance~1{schedule_maintenance_id}/delete
func (c *Client) DeleteScheduledMaintenance(ctx context.Context, id string) error {
	return c.deleteJSON(ctx, "/schedule-maintenance/"+id, nil)
}

// ListScheduledMaintenances returns scheduled maintenance windows matching
// query filters. Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1schedule-maintenance/get
func (c *Client) ListScheduledMaintenances(ctx context.Context, request ListScheduledMaintenancesRequest) (*ListScheduledMaintenancesResponse, error) {
	if err := validateQuery(request); err != nil {
		return nil, err
	}
	var response ListScheduledMaintenancesResponse
	if err := c.getJSON(ctx, "/schedule-maintenance", request.query(), &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// GetScheduledMaintenance finds a scheduled maintenance window by ID using
// ListScheduledMaintenances; see ListScheduledMaintenances for the source API
// reference.
func (c *Client) GetScheduledMaintenance(ctx context.Context, id string, monitorID string) (*ScheduledMaintenance, error) {
	for page := 1; ; page++ {
		request := ListScheduledMaintenancesRequest{PaginationRequest: PaginationRequest{Page: page, PerPage: 100}, MonitorID: monitorID}
		response, err := c.ListScheduledMaintenances(ctx, request)
		if err != nil {
			return nil, err
		}
		for _, maintenance := range response.ScheduledMaintenances {
			if maintenance.ID == id {
				return &maintenance, nil
			}
		}
		if response.Meta.Pagination.Next == nil || page >= response.Meta.Pagination.Last {
			return nil, nil
		}
	}
}
