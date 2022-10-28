// Package api QMS
//
// Documentation of the QMS Api
//
//	Schemes: http
//	BasePath: /
//	Version: V1
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
// swagger:meta
package swagger

import (
	"github.com/cyverse/QMS/internal/controllers"
	"github.com/cyverse/QMS/internal/httpmodel"
	"github.com/cyverse/QMS/internal/model"
)

// Note: the comments in this package don't conform to the convention of including the name of the entity that the
// comment describes. The reason for this is because the comments appear as-is in the API documentation. Confusing
// documentation is produced when the structure names appear in the API documentation.

// Error
//
// Having the same object definition for multiple HTTP response status codes seems to confuse ReDoc, so we're using
// aliases as a workaround.
//
// swagger:response errorResponse
type ErrorResponse struct {

	// in: body
	Body struct {

		// A brief description of the error
		Error string `json:"error"`

		// The status of the request
		Status string `json:"status"`
	}
}

// Bad Request
//
// swagger:response badRequestResponse
type BadRequestResponse struct {
	ErrorResponse
}

// Not Found
//
// swagger:response notFoundResponse
type NotFoundResponse struct {
	ErrorResponse
}

// Conflict
//
// swagger:response conflictResponse
type ConflictResponse struct {
	ErrorResponse
}

// Internal Server Error
//
// swagger:response internalServerErrorResponse
type InternalServerErrorResponse struct {
	ErrorResponse
}

// Documentation for the successful response body wrapper. The `Error` field could be included here as well, but it's
// being omitted for now simply because it produces less confusing documentation when the erorr and success response
// bodies are treated separately.
//
// swagger:model
type ResponseBodyWrapper struct {

	// The status of the request
	Status string `json:"status"`
}

// Service Information
//
// swagger:response rootResponse
type RootResponseWrapper struct {

	// in:body
	Body struct {
		ResponseBodyWrapper

		// The service information
		Result model.RootResponse `json:"result"`
	}
}

// Service API Version Information
//
// swagger:response apiVersionResponse
type APIVersionResponseWrapper struct {

	// in:body
	Body struct {
		ResponseBodyWrapper

		// The API version information
		Result model.APIVersionResponse `json:"result"`
	}
}

// General Success Message
//
// swagger:response successMessageResponse
type SuccessMessageResponseWrapoper struct {

	// in:body
	Body struct {
		ResponseBodyWrapper

		// The success message.
		Result string `json:"result"`
	}
}

// Parameters for the endpoint used to add multiple subscriptions.
//
// swagger:parameters addSubscriptions
type AddSubscriptionsParameters struct {

	// The subscriptions to add
	//
	// in: body
	Body model.SubscriptionRequests
}

// Subscriptions Response
//
// swagger:response subscriptionsResponse
type SubscriptionsResponse struct {

	// in:body
	Body struct {
		ResponseBodyWrapper

		// The list of subscription responses
		Result []model.SubscriptionResponse
	}
}

// Plan Listing
//
// swagger:response plansResponse
type PlansResponseWrapper struct {

	// in: body
	Body struct {
		ResponseBodyWrapper

		// The list of plans
		Result []model.Plan `json:"result"`
	}
}

// Plan ID
//
// swagger:parameters getPlanByID
type PlanIDParameter struct {

	// The plan identifier
	//
	// in:path
	// required:true
	PlanID string `json:"plan_id"`
}

// Plan Information
//
// swagger:response planResponse
type PlanResponseWrapper struct {

	// in: body
	Body struct {
		ResponseBodyWrapper

		// The plan information
		Result model.Plan `json:"result"`
	}
}

type AddPlans struct {
	// The Add Plans information
	//
	// in: body
	Body httpmodel.NewPlan
}

type AddPlanQuotaDefaults struct {
	// The planQuotaDefault information
	//
	// in: body
	Body controllers.PlanQuotaDefaultValues
}

type QuotaResponseWrapper struct {
	// The quota information
	//
	//in: body
	Body controllers.QuotaReq
}

// Users

// User Listing
//
// swagger:response userListing
type UserListingResponseWrapper struct {

	// in: body
	Body struct {
		ResponseBodyWrapper

		// The user listing
		Result []model.User `json:"result"`
	}
}

// Resource Types

// Resource Type Listing
//
// swagger:response resourceTypeListing
type ResourceTypeListingWrapper struct {

	// in: body
	Body struct {
		ResponseBodyWrapper

		// The resource type listing
		Result []model.ResourceType `json:"result"`
	}
}

// Resource Type Details
//
// swagger:response resourceTypeDetails
type ResourceTypeDetailsResponseWrapper struct {

	// in: body
	Body struct {
		ResponseBodyWrapper

		// The resource type information
		Result model.ResourceType `json:"result"`
	}
}

// Parameters for the endpoint used to add resource types.
//
// swagger:parameters addResourceType
type AddResourceTypeParameters struct {

	// The resource type information
	//
	// in: body
	Body model.ResourceType
}

// Parameters for the endpoint used to get resource type details.
//
// swagger:parameters getResourceTypeDetails
type GetResourceTypeDetailsParameters struct {

	// The resource type ID
	//
	// in: path
	// required: true
	ResourceTypeID string `json:"resource_type_id"`
}

// Parameters for the endpoint used to update resource types.
//
// swagger:parameters updateResourceType
type UpdateResourceTypeParameters struct {

	// The resource type ID
	//
	// in: path
	// required: true
	ResourceTypeID string `json:"resource_type_id"`

	// The resource type details
	//
	// in: body
	Body model.ResourceType
}
