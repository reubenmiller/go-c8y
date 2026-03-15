package jsonmodels

import "github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"

// ApplicationVersion represents an application version
type ApplicationVersion struct {
	jsondoc.Facade
}

func NewApplicationVersion(data []byte) ApplicationVersion {
	return ApplicationVersion{jsondoc.Facade{JSONDoc: jsondoc.New(data)}}
}

func (v ApplicationVersion) Version() string {
	return v.Get("version").String()
}

func (v ApplicationVersion) BinaryID() string {
	return v.Get("binaryId").String()
}

func (v ApplicationVersion) Tags() []string {
	tags := v.Get("tags").Array()
	result := make([]string, len(tags))
	for i, t := range tags {
		result[i] = t.String()
	}
	return result
}
