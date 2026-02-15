package model

// User data model
type User struct {
	ID       string `json:"id,omitempty"`
	Self     string `json:"self,omitempty"`
	Username string `json:"userName,omitempty"`
	Password string `json:"password,omitempty"`

	// The user's display name in Cumulocity
	DisplayName      string              `json:"displayName,omitempty"`
	FirstName        string              `json:"firstName,omitempty"`
	LastName         string              `json:"lastName,omitempty"`
	Phone            string              `json:"phone,omitempty"`
	Email            string              `json:"email,omitempty"`
	Newsletter       *bool               `json:"newsletter,omitempty"`
	Enabled          bool                `json:"enabled,omitempty"`
	CustomProperties any                 `json:"customProperties,omitempty"`
	Groups           *UserGroupReference `json:"groups,omitempty"`
	// Roles             *RoleReferenceCollection  `json:"roles,omitempty"`
	DevicePermissions map[string]any `json:"devicePermissions,omitempty"`
	EffectiveRoles    []UserRole     `json:"effectiveRoles,omitempty"`

	TwoFactorAuthenticationEnabled bool `json:"twoFactorAuthenticationEnabled,omitzero"`
	ShouldResetPassword            bool `json:"shouldResetPassword,omitzero"`
}

// UserCollection collection of users
type UserCollection struct {
	*BaseResponse

	Users []User `json:"users"`
}

type UserReferencesCollection struct {
	*BaseResponse

	References []UserReference `json:"references"`
}

func (c *UserReferencesCollection) Users() []User {
	users := make([]User, 0, len(c.References))
	for _, ref := range c.References {
		users = append(users, ref.User)
	}
	return users
}

type UserReference struct {
	Self string `json:"self,omitempty"`
	User User   `json:"user"`
}

func NewUserReference(user User) *UserReference {
	return &UserReference{
		User: user,
	}
}
