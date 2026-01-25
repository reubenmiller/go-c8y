package model

type UserRoleReference struct {
	Self string   `json:"self,omitempty"`
	Role UserRole `json:"role"`
}

type UserRole struct {
	Self string `json:"self,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type UserRoleReferenceCollection struct {
	Self       string              `json:"self,omitempty"`
	References []UserRoleReference `json:"references,omitempty"`
}

type UserRoleCollection struct {
	*BaseResponse

	Roles []UserRole `json:"roles"`
}
