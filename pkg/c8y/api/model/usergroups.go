package model

type UserGroupReference struct {
	Self  string    `json:"self,omitempty"`
	Group UserGroup `json:"group,omitempty"`
}

type UserGroupReferenceCollection struct {
	Self       string               `json:"self,omitempty"`
	References []UserGroupReference `json:"References,omitempty"`
}

type UserGroup struct {
	ID                uint64                      `json:"id,omitempty"`
	Self              string                      `json:"self,omitempty"`
	Name              string                      `json:"name,omitempty"`
	Roles             UserRoleReferenceCollection `json:"roles,omitempty"`
	DevicePermissions map[string]any              `json:"devicePermissions,omitempty"`
}

type UserGroupCollection struct {
	*BaseResponse

	Groups []UserGroup `json:"groups"`
}
