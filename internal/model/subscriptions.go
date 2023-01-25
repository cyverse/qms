package model

// SubscriptionRequest represents a request for a single subscription.
//
// swagger: model
type SubscriptionRequest struct {
	// The username to associate with the subscription
	//
	// required: true
	Username *string `json:"username"`

	// The name of the plan associated with the subscription
	//
	// required: true
	PlanName *string `json:"plan_name"`

	// True if the user paid for the subscription.
	Paid *bool `json:"paid"`
}

// SubscriptionRequests represents a list of subscription requests.
//
// swagger: model
type SubscriptionRequests struct {
	// The list of subscriptions to create
	//
	// required: true
	Subscriptions []SubscriptionRequest `json:"subscriptions"`
}

// SubscriptionResponse represents a response to a request for a single subscription.
//
// swagger: model
type SubscriptionResponse struct {
	Subscription

	// The reason the subscription couldn't be created if an error occurred.
	FailureReason *string `json:"failure_reason,omitempty"`

	// True if the subscription was just created.
	NewSubscription bool `json:"new_subscription"`
}

// SubscriptionResponseFromSubscription converts a user plan to a subscription response.
func SubscriptionResponseFromSubscription(subscription *Subscription, newSubscription bool) *SubscriptionResponse {
	var resp SubscriptionResponse
	resp.Subscription = *subscription
	resp.NewSubscription = newSubscription
	return &resp
}

// SubscriptionListing represents a list of subscriptions.
//
// swagger: model
type SubscriptionListing struct {
	// The subscriptions in the listing.
	Subscriptions []*Subscription `json:"subscriptions"`

	// The total number of matched subscriptions.
	Total int64 `json:"total"`
}
