package hetrixtools

import "context"

type (
	// ListContactListsResponse is returned by ListContactLists.
	ListContactListsResponse struct {
		// ContactLists contains the returned contact lists.
		ContactLists []ContactList `json:"contact_lists"`
		// Meta contains pagination and result-count metadata.
		Meta Meta `json:"meta"`
	}

	// ContactList describes a HetrixTools contact list.
	ContactList struct {
		// ID is the unique contact list ID.
		ID string `json:"id"`
		// Name is the contact list name.
		Name string `json:"name"`
		// Default reports whether this is the default contact list.
		Default bool `json:"default"`
		// Email contains email recipients in this contact list.
		Email []string `json:"email"`
		// PhoneSMS contains SMS recipients in this contact list.
		PhoneSMS []string `json:"phone_sms"`
		// Telegram contains Telegram usernames in this contact list.
		Telegram []string `json:"telegram"`
		// Pushbullet contains Pushbullet usernames in this contact list.
		Pushbullet []string `json:"pushbullet"`
		// Pushover contains Pushover integration settings.
		Pushover *ContactListPushover `json:"pushover"`
		// Twitter contains deprecated Twitter usernames in this contact list.
		Twitter []string `json:"twitter"`
		// Slack contains Slack integration settings.
		Slack *ContactListWebhookTarget `json:"slack"`
		// Discord contains Discord integration settings.
		Discord *ContactListWebhookTarget `json:"discord"`
		// MattermostRocketchat contains Mattermost or Rocket.Chat integration settings.
		MattermostRocketchat *ContactListWebhookTarget `json:"mattermost_rocketchat"`
		// MicrosoftTeams contains Microsoft Teams integration settings.
		MicrosoftTeams *ContactListWebhookOnly `json:"microsoft_teams"`
		// PagerDuty contains PagerDuty integration settings.
		PagerDuty *ContactListKeyOnly `json:"pagerduty"`
		// Opsgenie contains Opsgenie integration settings.
		Opsgenie *ContactListPriorityKey `json:"opsgenie"`
		// VictorOps contains VictorOps integration settings.
		VictorOps *ContactListVictorOps `json:"victorops"`
		// Webhook contains generic webhook integration settings.
		Webhook *ContactListWebhook `json:"webhook"`
		// DND contains do-not-disturb schedule entries.
		DND []ContactListDND `json:"dnd"`
	}

	// ContactListPushover describes Pushover contact-list settings.
	ContactListPushover struct {
		// Key is the Pushover API key.
		Key string `json:"key"`
		// Priority is the Pushover priority.
		Priority int `json:"priority"`
	}

	// ContactListWebhookTarget describes webhook settings with a target and target-hiding flag.
	ContactListWebhookTarget struct {
		// Webhook is the destination webhook URL.
		Webhook string `json:"webhook"`
		// Target is the destination channel or target name.
		Target string `json:"target"`
		// HideTarget reports whether HetrixTools hides the monitored target in notifications.
		HideTarget bool `json:"hide_target"`
	}

	// ContactListWebhookOnly describes webhook settings that only contain a webhook URL.
	ContactListWebhookOnly struct {
		// Webhook is the destination webhook URL.
		Webhook string `json:"webhook"`
	}

	// ContactListKeyOnly describes integration settings that only contain a key.
	ContactListKeyOnly struct {
		// Key is the integration key.
		Key string `json:"key"`
	}

	// ContactListPriorityKey describes integration settings with a key and priority.
	ContactListPriorityKey struct {
		// Key is the integration key.
		Key string `json:"key"`
		// Priority is the integration priority.
		Priority string `json:"priority"`
	}

	// ContactListVictorOps describes VictorOps contact-list settings.
	ContactListVictorOps struct {
		// Key is the VictorOps API key.
		Key string `json:"key"`
		// Route is the VictorOps route.
		Route string `json:"route"`
		// Priority is the VictorOps priority.
		Priority string `json:"priority"`
	}

	// ContactListWebhook describes generic webhook contact-list settings.
	ContactListWebhook struct {
		// URL is the webhook URL.
		URL string `json:"url"`
		// Authentication is the webhook authentication method.
		Authentication string `json:"authentication"`
	}

	// ContactListDND describes one contact-list do-not-disturb window.
	ContactListDND struct {
		// Day is the day of the week.
		Day string `json:"day"`
		// Hour is the hour of the day.
		Hour string `json:"hour"`
	}

	// ListContactListsRequest filters contact-list list results.
	ListContactListsRequest struct {
		// PaginationRequest contains page and per_page filters. Contact lists accept per_page up to 200.
		PaginationRequest
	}
)

func (r ListContactListsRequest) query() map[string]string {
	values := map[string]string{}
	r.PaginationRequest.appendQuery(values)
	return values
}

// ListContactLists returns HetrixTools contact lists matching query filters.
// Source-of-truth API docs:
//
//   - https://docs.hetrixtools.com/api/v3/#/paths/~1contact-lists/get
func (c *Client) ListContactLists(ctx context.Context, request ListContactListsRequest) (*ListContactListsResponse, error) {
	if err := validateQuery(request); err != nil {
		return nil, err
	}
	var response ListContactListsResponse
	if err := c.getJSON(ctx, "/contact-lists", request.query(), &response); err != nil {
		return nil, err
	}
	return &response, nil
}
