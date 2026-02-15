package model

// RetentionRule data model
type RetentionRule struct {
	// RetentionRule id
	ID string `json:"id,omitempty"`

	// RetentionRule will be applied to documents with source
	Source string `json:"source,omitempty"`

	// RetentionRule will be applied to documents with type
	Type string `json:"type,omitempty"`

	// RetentionRule will be applied to this type of documents, possible values [ALARM, AUDIT, EVENT, MEASUREMENT, OPERATION, *]
	DataType string `json:"dataType,omitempty"`

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
