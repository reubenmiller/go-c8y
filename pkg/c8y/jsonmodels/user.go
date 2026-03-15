package jsonmodels

import (
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

type User struct {
	jsondoc.Facade
}

func NewUser(b []byte) User {
	return User{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (u User) ID() string {
	return u.Get("id").String()
}

func (u User) UserName() string {
	return u.Get("userName").String()
}

func (u User) Email() string {
	return u.Get("email").String()
}

func (u User) FirstName() string {
	return u.Get("firstName").String()
}

func (u User) LastName() string {
	return u.Get("lastName").String()
}

func (u User) Enabled() bool {
	return u.Get("enabled").Bool()
}

func (u User) Self() string {
	return u.Get("self").String()
}

func (u User) LastPasswordChange() time.Time {
	return u.Get("lastPasswordChange").Time()
}
