package model

// ResourceType defines the structure for ResourceTypes.
//
// swagger:model
type ResourceType struct {
	// The resource type ID
	//
	// readOnly: true
	ID *string `gorm:"type:uuid;default:uuid_generate_v1()" json:"id"`
	// The resource type name
	//
	// required: true
	Name string `gorm:"not null;unique" json:"name"`
	// The unit of measure used for the resource type
	//
	// required: true
	Unit string `gorm:"not null;unique" json:"unit"`
}

var ResourceTypeNames = []string{
	"cpu.hours",
	"data.size",
}

var ResourceTypeUnits = []string{
	"cpu hours",
	"bytes",
}
