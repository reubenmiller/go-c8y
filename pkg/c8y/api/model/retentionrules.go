package model

// RetentionRuleDataType represents the type of document a retention rule applies to
type RetentionRuleDataType string

const (
	RetentionRuleDataTypeAlarm       RetentionRuleDataType = "ALARM"
	RetentionRuleDataTypeAudit       RetentionRuleDataType = "AUDIT"
	RetentionRuleDataTypeEvent       RetentionRuleDataType = "EVENT"
	RetentionRuleDataTypeMeasurement RetentionRuleDataType = "MEASUREMENT"
	RetentionRuleDataTypeOperation   RetentionRuleDataType = "OPERATION"
	RetentionRuleDataTypeAll         RetentionRuleDataType = "*"
)

// RetentionRule data model
type RetentionRule struct {
	// RetentionRule id
	ID string `json:"id,omitempty"`

	// RetentionRule will be applied to documents with source
	Source string `json:"source,omitempty"`

	// RetentionRule will be applied to documents with type
	Type string `json:"type,omitempty"`

	// RetentionRule will be applied to this type of documents, possible values [ALARM, AUDIT, EVENT, MEASUREMENT, OPERATION, *]
	DataType RetentionRuleDataType `json:"dataType,omitempty"`

	// RetentionRule will be applied to documents with fragmentType
	FragmentType string `json:"fragmentType,omitempty"`

	// Link to this resource
	Self string `json:"self,omitempty"`

	// Maximum age of document in days
	MaximumAge int64 `json:"maximumAge,omitempty"`

	// Whether the rule is editable. Can be updated only by management tenant
	Editable bool `json:"editable,omitempty"`
}

// RetentionRuleCollection collection of retention rules
type RetentionRuleCollection struct {
	*BaseResponse

	RetentionRules []RetentionRule `json:"retentionRules"`
}
