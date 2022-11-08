package model

const (
	RESOURCE_TYPE_CPU_HOURS = "cpu.hours"
	RESOURCE_TYPE_DATA_SIZE = "data.size"
)

// ResourceType defines the structure for ResourceTypes.
//
// swagger:model
type ResourceType struct {
	// The resource type ID
	//
	// readOnly: true
	ID *string `gorm:"type:uuid;default:uuid_generate_v1()" json:"id,omitempty"`
	// The resource type name
	//
	// required: true
	Name string `gorm:"not null;unique" json:"name,omitempty"`
	// The unit of measure used for the resource type
	//
	// required: true
	Unit string `gorm:"not null;unique" json:"unit,omitempty"`
}
