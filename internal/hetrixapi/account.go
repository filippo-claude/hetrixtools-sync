package hetrixtools

import "context"

type (
	// AccountLimitsResponse is returned by GetAccountLimits.
	AccountLimitsResponse struct {
		// Uptime contains uptime monitor usage and limits.
		Uptime AccountUptimeLimits `json:"uptime"`
		// Blacklist contains blacklist monitor and API check usage and limits.
		Blacklist AccountBlacklistLimits `json:"blacklist"`
		// SubAccounts contains sub-account usage and limits.
		SubAccounts AccountUsageLimit `json:"sub_accounts"`
		// SMSCredits contains SMS credit usage, limits, and credit source details.
		SMSCredits AccountCreditUsageLimit `json:"sms_credits"`
		// AccountCredit contains prepaid account credit balance details.
		AccountCredit AccountCreditBalance `json:"account_credit"`
		// APIV1V2 contains API v1/v2 usage and limits.
		APIV1V2 AccountUsageLimit `json:"api_v1_v2"`
	}

	// AccountUptimeLimits contains uptime-specific account limits.
	AccountUptimeLimits struct {
		// Monitors contains uptime monitor usage and limits.
		Monitors AccountUsageLimit `json:"monitors"`
	}

	// AccountBlacklistLimits contains blacklist-specific account limits.
	AccountBlacklistLimits struct {
		// Monitors contains blacklist monitor usage and limits.
		Monitors AccountUsageLimit `json:"monitors"`
		// APICheckCredits contains blacklist API check credit usage and limits.
		APICheckCredits AccountCreditUsageLimit `json:"api_check_credits"`
	}

	// AccountUsageLimit contains a usage count and corresponding limit.
	AccountUsageLimit struct {
		// Usage is the current amount used.
		Usage int64 `json:"usage"`
		// Limit is the maximum amount allowed.
		Limit int64 `json:"limit"`
	}

	// AccountCreditUsageLimit contains a usage/limit pair with credit source details.
	AccountCreditUsageLimit struct {
		// Usage is the current amount used.
		Usage int64 `json:"usage"`
		// Limit is the maximum amount available.
		Limit int64 `json:"limit"`
		// Details breaks down the source of the available credits.
		Details AccountCreditUsageDetails `json:"details"`
	}

	// AccountCreditUsageDetails breaks down included and purchased credits.
	AccountCreditUsageDetails struct {
		// MonthlyFromPlans is the number of monthly credits included in active plans.
		MonthlyFromPlans int64 `json:"monthly_from_plans"`
		// ExtraCredits is the number of extra purchased credits.
		ExtraCredits int64 `json:"extra_credits"`
	}

	// AccountCreditBalance contains the prepaid account credit balance.
	AccountCreditBalance struct {
		// Balance is the current prepaid account credit balance.
		Balance float64 `json:"balance"`
	}
)

// GetAccountLimits returns the current account-level HetrixTools limits.
// Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1account~1limits/get
func (c *Client) GetAccountLimits(ctx context.Context) (*AccountLimitsResponse, error) {
	var response AccountLimitsResponse
	if err := c.getJSON(ctx, "/account/limits", nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}
