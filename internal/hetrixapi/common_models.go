package hetrixtools

type (
	// ActionResponse is the successful response returned by HetrixTools mutation endpoints.
	ActionResponse struct {
		// Status is the API status string, such as SUCCESS.
		Status string `json:"status"`
		// MonitorID is the monitor identifier returned by monitor create operations.
		MonitorID string `json:"monitor_id"`
		// ServerID is the server-agent identifier returned by heartbeat operations.
		ServerID string `json:"server_id"`
		// Action describes the mutation performed when the API returns one.
		Action string `json:"action"`
	}

	// APIErrorResponse is the error envelope returned by HetrixTools mutation endpoints.
	APIErrorResponse struct {
		// Status is the API status string, such as ERROR.
		Status string `json:"status"`
		// ErrorMessage contains the API-provided error detail.
		ErrorMessage string `json:"error_message"`
	}

	// PaginationRequest contains common HetrixTools paginated-list parameters.
	PaginationRequest struct {
		// Page is the result page to request. HetrixTools pages are one-based and list endpoints accept pages 1 through 10000.
		Page int `validate:"omitempty,min=1,max=10000"`
		// PerPage is the number of results per page. Endpoint-specific request validation enforces each API's documented maximum.
		PerPage int `validate:"omitempty,min=1"`
	}

	// Pagination describes HetrixTools paginated list metadata.
	Pagination struct {
		// Current is the current page number.
		Current int `json:"current"`
		// Last is the last page number.
		Last int `json:"last"`
		// Previous is the previous page number when available.
		Previous *int `json:"previous"`
		// Next is the next page number when available.
		Next *int `json:"next"`
	}

	// Meta contains HetrixTools list response metadata.
	Meta struct {
		// Total is the total item count reported by the API.
		Total int `json:"total"`
		// TotalFiltered is the count after API-side filters are applied.
		TotalFiltered int `json:"total_filtered"`
		// Returned is the number of items in the current response.
		Returned int `json:"returned"`
		// Pagination contains page navigation metadata.
		Pagination Pagination `json:"pagination"`
	}
)
