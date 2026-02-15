package model

// TenantOption representation
type TenantOption struct {
	Self     string `json:"self,omitempty"`
	Category string `json:"category,omitempty"`
	Key      string `json:"key,omitempty"`
	Value    string `json:"value,omitempty"`
}

// TenantOptionCollection collection of tenant options
type TenantOptionCollection struct {
	*BaseResponse

	Options []TenantOption `json:"options"`
}
