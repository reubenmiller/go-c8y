package jsonmodels

import "github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"

type TenantOption struct {
	jsondoc.JSONDoc
}

func NewTenantOption(b []byte) TenantOption {
	return TenantOption{jsondoc.New(b)}
}

// Self returns the URL to this resource
func (o TenantOption) Self() string {
	return o.Get("self").String()
}

// Category returns the category of the tenant option
func (o TenantOption) Category() string {
	return o.Get("category").String()
}

// Key returns the key of the tenant option
func (o TenantOption) Key() string {
	return o.Get("key").String()
}

// Value returns the value of the tenant option
func (o TenantOption) Value() string {
	return o.Get("value").String()
}
