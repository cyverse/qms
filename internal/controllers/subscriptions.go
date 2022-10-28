package controllers

import "github.com/labstack/echo/v4"

// AddSubscriptions creates the subscriptions described in the request body.
//
// swagger:route POST /v1/subscriptions subscriptions addSubscriptions
//
// # Add Subscriptions
//
// Creates the subscriptions described in the request body.
//
// Responses:
//
//	200: subscriptionsResponse
func (s Server) AddSubscriptions(ctx echo.Context) error {
	var err error

	return err
}
