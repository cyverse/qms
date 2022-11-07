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
}

//	SubscriptionRequests represents a list of subscription requests.
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
	UserPlan

	// The reason the subscription couldn't be created if an error occurred.
	FailureReason *string `json:"failure_reason"`
}

// SubscriptionResponseFromUserPlan converts a user plan to a subscription response.
func SubscriptionResponseFromUserPlan(userPlan *UserPlan) *SubscriptionResponse {
	var resp SubscriptionResponse
	resp.UserPlan = *userPlan
	return &resp
}
