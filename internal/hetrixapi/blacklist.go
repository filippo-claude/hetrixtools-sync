package hetrixtools

import "context"

type (
	// ListBlacklistsResponse is returned by ListBlacklists.
	ListBlacklistsResponse struct {
		// IPv4 contains IPv4 RBL entries checked by HetrixTools.
		IPv4 []BlacklistProvider `json:"ipv4"`
		// Domains contains domain RBL entries checked by HetrixTools.
		Domains []BlacklistProvider `json:"domains"`
	}

	// BlacklistProvider describes one RBL provider checked by HetrixTools.
	BlacklistProvider struct {
		// ID is the unique RBL entry identifier.
		ID string `json:"id"`
		// RBL is the Real-time Blackhole List name.
		RBL string `json:"rbl"`
		// Optional reports whether the RBL is optional and opt-in.
		Optional bool `json:"optional"`
		// Ignored reports whether this RBL is ignored.
		Ignored bool `json:"ignored"`
	}
)

// ListBlacklists returns the blacklist providers monitored by HetrixTools.
// Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1blacklists/get
func (c *Client) ListBlacklists(ctx context.Context) (*ListBlacklistsResponse, error) {
	var response ListBlacklistsResponse
	if err := c.getJSON(ctx, "/blacklists", nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}
