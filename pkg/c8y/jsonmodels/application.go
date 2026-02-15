package jsonmodels

import (
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
)

type Application struct {
	jsondoc.Facade
}

func NewApplication(b []byte) Application {
	return Application{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (a Application) ID() string {
	return a.Get("id").String()
}

func (a Application) Name() string {
	return a.Get("name").String()
}

func (a Application) Key() string {
	return a.Get("key").String()
}

func (a Application) Type() string {
	return a.Get("type").String()
}

func (a Application) Availability() string {
	return a.Get("availability").String()
}

func (a Application) Self() string {
	return a.Get("self").String()
}

func (a Application) CreationTime() time.Time {
	return a.Get("creationTime").Time()
}

func (a Application) LastUpdated() time.Time {
	return a.Get("lastUpdated").Time()
}
