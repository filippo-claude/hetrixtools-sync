package hetrixtools

import "context"

type (
	// ServerAgentResponse describes the server agent attached to an uptime monitor.
	ServerAgentResponse struct {
		// AgentID is the attached server-agent ID. It is nil when no agent is attached.
		AgentID *string `json:"agent_id"`
	}

	// ServerAgentWarningPoliciesResponse is returned by GetServerAgentWarningPolicies.
	ServerAgentWarningPoliciesResponse struct {
		ServerAgentWarningPolicies
	}

	// ServerAgentWarningPolicies contains server-agent warning policy configuration.
	ServerAgentWarningPolicies struct {
		// AgentDataWarn issues warnings when the server monitoring agent stops sending data.
		AgentDataWarn ServerAgentDataWarningPolicy `json:"agent_data_warn"`
		// CPUUsageWarn issues warnings when CPU usage reaches a threshold.
		CPUUsageWarn ServerAgentUsageWarningPolicy `json:"cpu_usage_warn"`
		// IOWaitUsageWarn issues warnings when iowait usage reaches a threshold.
		IOWaitUsageWarn ServerAgentUsageWarningPolicy `json:"iowait_usage_warn"`
		// RAMUsageWarn issues warnings when RAM usage reaches a threshold.
		RAMUsageWarn ServerAgentUsageWarningPolicy `json:"ram_usage_warn"`
		// SwapUsageWarn issues warnings when swap usage reaches a threshold.
		SwapUsageWarn ServerAgentUsageWarningPolicy `json:"swap_usage_warn"`
		// DiskUsageWarn issues warnings when disk usage reaches a threshold.
		DiskUsageWarn ServerAgentDiskWarningPolicy `json:"disk_usage_warn"`
		// RAIDWarn issues warnings when RAID status is not completely healthy.
		RAIDWarn ServerAgentRAIDWarningPolicy `json:"raid_warn"`
		// DriveErrorsWarn issues warnings when drive errors reach a threshold.
		DriveErrorsWarn ServerAgentUsageWarningPolicy `json:"drive_errors_warn"`
		// DriveWearoutWarn issues warnings when drive wearout reaches a threshold.
		DriveWearoutWarn ServerAgentUsageWarningPolicy `json:"drive_wearout_warn"`
		// DriveSMARTWarn issues warnings when S.M.A.R.T. tests fail.
		DriveSMARTWarn ServerAgentSimpleWarningPolicy `json:"drive_smart_warn"`
		// NetworkInWarn issues warnings when inbound network usage reaches a threshold.
		NetworkInWarn ServerAgentNetworkWarningPolicy `json:"network_in_warn"`
		// NetworkOutWarn issues warnings when outbound network usage reaches a threshold.
		NetworkOutWarn ServerAgentNetworkWarningPolicy `json:"network_out_warn"`
		// ServicesWarn issues warnings when monitored services go down.
		ServicesWarn ServerAgentServicesWarningPolicy `json:"services_warn"`
	}

	// ServerAgentDataWarningPolicy configures missing-agent-data warnings.
	ServerAgentDataWarningPolicy struct {
		// Enabled reports whether this policy is enabled.
		Enabled bool `json:"enabled"`
		// Threshold is the number of consecutive minutes without agent data required to trigger a warning.
		Threshold int64 `json:"threshold"`
		// WarnFrequency is the minimum number of minutes between issued warnings.
		WarnFrequency int64 `json:"warn_frequency"`
	}

	// ServerAgentUsageWarningPolicy configures threshold-based usage warnings.
	ServerAgentUsageWarningPolicy struct {
		// Enabled reports whether this policy is enabled.
		Enabled bool `json:"enabled"`
		// Threshold is the usage threshold that triggers a warning.
		Threshold int64 `json:"threshold"`
		// TimeFrame is the number of recent minutes over which usage is averaged.
		TimeFrame int64 `json:"time_frame"`
		// WarnFrequency is the minimum number of minutes between issued warnings.
		WarnFrequency int64 `json:"warn_frequency"`
	}

	// ServerAgentDiskWarningPolicy configures disk usage warnings.
	ServerAgentDiskWarningPolicy struct {
		// Enabled reports whether this policy is enabled.
		Enabled bool `json:"enabled"`
		// Threshold is the usage threshold that triggers a warning.
		Threshold int64 `json:"threshold"`
		// TimeFrame is the number of recent minutes over which usage is averaged.
		TimeFrame int64 `json:"time_frame"`
		// WarnFrequency is the minimum number of minutes between issued warnings.
		WarnFrequency int64 `json:"warn_frequency"`
		// WarnDisks contains disk-specific warning toggles.
		WarnDisks []ServerAgentWarnDisk `json:"warn_disks"`
	}

	// ServerAgentWarnDisk describes one disk's warning settings.
	ServerAgentWarnDisk struct {
		// ID is the unique disk identifier.
		ID string `json:"id"`
		// Enabled reports whether this disk generates usage warnings.
		Enabled bool `json:"enabled"`
		// Mount is the disk mount point.
		Mount string `json:"mount"`
	}

	// ServerAgentRAIDWarningPolicy configures RAID status warnings.
	ServerAgentRAIDWarningPolicy struct {
		// Enabled reports whether this policy is enabled.
		Enabled bool `json:"enabled"`
		// WarnType is the RAID severity that triggers warnings, either not_ideal or critical.
		WarnType string `json:"warn_type"`
		// WarnFrequency is the minimum number of minutes between issued warnings.
		WarnFrequency int64 `json:"warn_frequency"`
	}

	// ServerAgentSimpleWarningPolicy configures warnings that only need enablement and frequency.
	ServerAgentSimpleWarningPolicy struct {
		// Enabled reports whether this policy is enabled.
		Enabled bool `json:"enabled"`
		// WarnFrequency is the minimum number of minutes between issued warnings.
		WarnFrequency int64 `json:"warn_frequency"`
	}

	// ServerAgentNetworkWarningPolicy configures network usage warnings.
	ServerAgentNetworkWarningPolicy struct {
		// Enabled reports whether this policy is enabled.
		Enabled bool `json:"enabled"`
		// Threshold is the network usage threshold that triggers a warning.
		Threshold int64 `json:"threshold"`
		// ThresholdType is the threshold unit: kbps, mbps, or gbps.
		ThresholdType string `json:"threshold_type"`
		// TimeFrame is the number of recent minutes over which usage is averaged.
		TimeFrame int64 `json:"time_frame"`
		// WarnFrequency is the minimum number of minutes between issued warnings.
		WarnFrequency int64 `json:"warn_frequency"`
		// WarnNICs contains network-interface-specific warning toggles.
		WarnNICs []ServerAgentWarnNIC `json:"warn_nics"`
	}

	// ServerAgentWarnNIC describes one network interface's warning settings.
	ServerAgentWarnNIC struct {
		// ID is the unique network interface identifier.
		ID string `json:"id"`
		// Enabled reports whether this interface generates usage warnings.
		Enabled bool `json:"enabled"`
		// Name is the network interface name.
		Name string `json:"name"`
	}

	// ServerAgentServicesWarningPolicy configures service-down warnings.
	ServerAgentServicesWarningPolicy struct {
		// Enabled reports whether this policy is enabled.
		Enabled bool `json:"enabled"`
		// WarnFrequency is the minimum number of minutes between issued warnings.
		WarnFrequency int64 `json:"warn_frequency"`
		// WarnServices contains service-specific warning toggles.
		WarnServices []ServerAgentWarnService `json:"warn_services"`
	}

	// ServerAgentWarnService describes one monitored service's warning settings.
	ServerAgentWarnService struct {
		// ID is the unique service identifier.
		ID string `json:"id"`
		// Enabled reports whether this service generates warnings.
		Enabled bool `json:"enabled"`
		// Name is the service name.
		Name string `json:"name"`
	}

	// ServerAgentWarningPoliciesRequest replaces server-agent warning policies.
	ServerAgentWarningPoliciesRequest struct {
		ServerAgentWarningPolicies
	}
)

// AttachServerAgent attaches a server agent to an uptime monitor.
// Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1uptime-monitors~1{monitor_id}~1server-agent/post
func (c *Client) AttachServerAgent(ctx context.Context, monitorID string) (*ServerAgentResponse, error) {
	var result ServerAgentResponse
	if err := c.postJSON(ctx, "/uptime-monitors/"+monitorID+"/server-agent", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetServerAgent returns the server agent attached to an uptime monitor.
// Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1uptime-monitors~1{monitor_id}~1server-agent/get
func (c *Client) GetServerAgent(ctx context.Context, monitorID string) (*ServerAgentResponse, error) {
	var result ServerAgentResponse
	if err := c.getJSON(ctx, "/uptime-monitors/"+monitorID+"/server-agent", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DetachServerAgent detaches a server agent from an uptime monitor.
// Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1uptime-monitors~1{monitor_id}~1server-agent/delete
func (c *Client) DetachServerAgent(ctx context.Context, monitorID string) error {
	return c.deleteJSON(ctx, "/uptime-monitors/"+monitorID+"/server-agent", nil)
}

// GetServerAgentWarningPolicies returns server-agent warning policies for an
// uptime monitor. Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1uptime-monitors~1{monitor_id}~1server-agent~1warning-policies/get
func (c *Client) GetServerAgentWarningPolicies(ctx context.Context, monitorID string) (*ServerAgentWarningPoliciesResponse, error) {
	var response ServerAgentWarningPoliciesResponse
	if err := c.getJSON(ctx, "/uptime-monitors/"+monitorID+"/server-agent/warning-policies", nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// UpdateServerAgentWarningPolicies replaces server-agent warning policies for
// an uptime monitor. Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1uptime-monitors~1{monitor_id}~1server-agent~1warning-policies/put
func (c *Client) UpdateServerAgentWarningPolicies(ctx context.Context, monitorID string, request ServerAgentWarningPoliciesRequest) error {
	return c.putJSON(ctx, "/uptime-monitors/"+monitorID+"/server-agent/warning-policies", request, nil)
}
