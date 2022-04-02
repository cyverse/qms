package controllers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/cyverse-de/echo-middleware/v2/params"
	"github.com/cyverse/QMS/internal/model"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// extractResourceTypeID extracts and validates the resource type ID path parameter.
func extractResourceTypeID(ctx echo.Context) (string, error) {
	resourceTypeID, err := params.ValidatedPathParam(ctx, "resource_type_id", "uuid_rfc4122")
	if err != nil {
		return "", fmt.Errorf("the resource type ID must be a valid UUID")
	}
	return resourceTypeID, nil
}

// swagger:route GET /v1/resource-types resource-types listResourceTypes
//
// List Resource Types
//
// Lists all the resource types defined in the QMS database.
//
// responses:
//   200: resourceTypeListing
//   500: internalServerErrorResponse

// ListResourceTypes is the handler for the GET /v1/resource-types and GET /v1/resource-types endpoints.
func (s Server) ListResourceTypes(ctx echo.Context) error {
	var data []model.ResourceType
	err := s.GORMDB.Debug().Find(&data).Error
	if err != nil {
		msg := fmt.Sprintf("unable to list resource types: %s", err)
		return model.Error(ctx, msg, http.StatusInternalServerError)
	}
	return model.Success(ctx, data, http.StatusOK)
}

// swagger:route POST /v1/resource-types resource-types addResourceType
//
// Add Resource Type
//
// Adds a new resource type to the QMS database.
//
// responses:
//   200: resourceTypeDetails
//   400: badRequestResponse
//   409: conflictResponse
//   500: internalServerErrorResponse

// AddResourceType is the handler for the POST /v1/resource-types endpoint.
func (s Server) AddResourceType(ctx echo.Context) error {
	var err error

	//  Extract and validate the request body.
	var resourceType model.ResourceType
	if err = ctx.Bind(&resourceType); err != nil {
		msg := fmt.Sprintf("invalid request body: %s", err)
		return model.Error(ctx, msg, http.StatusBadRequest)
	}
	if resourceType.Name == "" || resourceType.Unit == "" {
		msg := "the resource type name and unit are both required"
		return model.Error(ctx, msg, http.StatusBadRequest)
	}

	// Guard against the case where the ID is specified in the request body, which breaks our duplicate check.
	resourceType.ID = nil

	// Save the resource type.
	result := s.GORMDB.
		Select("ID", "Name", "Unit").
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&resourceType)
	if result.Error != nil {
		msg := fmt.Sprintf("unable to save the resource type: %s", result.Error)
		return model.Error(ctx, msg, http.StatusInternalServerError)
	}

	// If the ID wasn't populated and an error didn't occur then there must have been a conflict.
	if resourceType.ID == nil || *resourceType.ID == "" {
		msg := fmt.Sprintf("a resource type with the given name already exists: %s", resourceType.Name)
		return model.Error(ctx, msg, http.StatusConflict)
	}

	return model.Success(ctx, resourceType, http.StatusOK)
}

// swagger:route GET /v1/resource-types/{resource_type_id} resource-types getResourceTypeDetails
//
// Get Resource Type Details
//
// Returns information about the resource type with the given identifier.
//
// responses:
//   200: resourceTypeDetails
//   400: badRequestResponse
//   404: notFoundResponse
//   500: internalServerErrorResponse

// GetResourceTypeDetails is the handler for the GET /v1/resource-types/{resource_type_id} endpoint.
func (s Server) GetResourceTypeDetails(ctx echo.Context) error {
	var err error

	// Extract and validate the resource type ID.
	resourceTypeID, err := extractResourceTypeID(ctx)
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusBadRequest)
	}

	// Look up the resource type.
	resourceType := model.ResourceType{ID: &resourceTypeID}
	err = s.GORMDB.Take(&resourceType).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		msg := fmt.Sprintf("resource type not found: %s", resourceTypeID)
		return model.Error(ctx, msg, http.StatusNotFound)
	} else if err != nil {
		msg := fmt.Sprintf("unable to look up the resource type: %s", err)
		return model.Error(ctx, msg, http.StatusInternalServerError)
	}

	return model.Success(ctx, &resourceType, http.StatusOK)
}

// swagger:route PUT /v1/resource-types/{resource_type_id} resource-types updateResourceType
//
// Update Resource Type
//
// Updates an existing resource type in the QMS database.
//
// responses:
//   200: resourceTypeDetails
//   400: badRequestResponse
//   404: notFoundResponse
//   500: internalServerErrorResponse

// UpdateResourceType is the handler for the PUT /v1/resource-types/{resource_type_id} endpoint.
func (s Server) UpdateResourceType(ctx echo.Context) error {
	context := ctx.Request().Context()
	var err error

	// Extract and validate the resource type ID.
	resourceTypeID, err := extractResourceTypeID(ctx)
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusBadRequest)
	}

	//  Extract and validate the request body.
	var inboundResourceType model.ResourceType
	if err = ctx.Bind(&inboundResourceType); err != nil {
		msg := fmt.Sprintf("invalid request body: %s", err)
		return model.Error(ctx, msg, http.StatusBadRequest)
	}
	if inboundResourceType.Name == "" || inboundResourceType.Unit == "" {
		msg := "the resource type name and unit are both required"
		return model.Error(ctx, msg, http.StatusBadRequest)
	}

	// Perform these steps in a transaction to ensure consistency.
	existingResourceType := model.ResourceType{ID: &resourceTypeID}
	err = s.GORMDB.Transaction(func(tx *gorm.DB) error {
		var err error

		// Verify that the resource type exists.
		err = tx.WithContext(context).Take(&existingResourceType).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			msg := fmt.Sprintf("resource type not found: %s", resourceTypeID)
			return echo.NewHTTPError(http.StatusNotFound, msg)
		} else if err != nil {
			msg := fmt.Sprintf("unable to look up the resource type: %s", err)
			return echo.NewHTTPError(http.StatusInternalServerError, msg)
		}

		// Update the resource type.
		existingResourceType.Name = inboundResourceType.Name
		existingResourceType.Unit = inboundResourceType.Unit
		err = tx.WithContext(context).Save(&existingResourceType).Error
		if err != nil {
			msg := fmt.Sprintf("unable to update the resource type: %s", err)
			return echo.NewHTTPError(http.StatusInternalServerError, msg)
		}

		return nil
	})
	if err != nil {
		return model.HTTPError(ctx, err.(*echo.HTTPError))
	}

	return model.Success(ctx, existingResourceType, http.StatusOK)
}
