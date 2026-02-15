package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsondoc"
	"github.com/tidwall/gjson"
)

type BulkNewDeviceRequests struct {
	jsondoc.Facade
}

func NewBulkNewDeviceRequests(b []byte) BulkNewDeviceRequests {
	return BulkNewDeviceRequests{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (i BulkNewDeviceRequests) Total() int64 {
	return i.Get("numberOfAll").Int()
}

func (i BulkNewDeviceRequests) TotalCreated() int64 {
	return i.Get("numberOfCreated").Int()
}

func (i BulkNewDeviceRequests) TotalFailed() int64 {
	return i.Get("numberOfFailed").Int()
}

func (i BulkNewDeviceRequests) TotalSuccessful() int64 {
	return i.Get("numberOfSuccessful").Int()
}

func (i BulkNewDeviceRequests) CredentialUpdatedList() []BulkNewDeviceRequestDetails {
	node := i.Get("credentialUpdatedList")

	if !node.IsArray() {
		return []BulkNewDeviceRequestDetails{}
	}

	results := make([]BulkNewDeviceRequestDetails, 0, len(node.Array()))
	node.ForEach(func(key, value gjson.Result) bool {
		results = append(results, NewBulkNewDeviceRequestDetails([]byte(value.Raw)))
		return true
	})
	return results
}

func (i BulkNewDeviceRequests) FailedCreationList() []BulkNewDeviceRequestDetails {
	node := i.Get("failedCreationList")

	if !node.IsArray() {
		return []BulkNewDeviceRequestDetails{}
	}

	results := make([]BulkNewDeviceRequestDetails, 0, len(node.Array()))
	node.ForEach(func(key, value gjson.Result) bool {
		results = append(results, NewBulkNewDeviceRequestDetails([]byte(value.Raw)))
		return true
	})
	return results
}

type BulkNewDeviceRequestDetails struct {
	jsondoc.Facade
}

func NewBulkNewDeviceRequestDetails(b []byte) BulkNewDeviceRequestDetails {
	return BulkNewDeviceRequestDetails{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (i BulkNewDeviceRequestDetails) Status() string {
	return i.Get("BulkNewDeviceStatus").String()
}

func (i BulkNewDeviceRequestDetails) DeviceID() string {
	return i.Get("deviceId").String()
}

func (i BulkNewDeviceRequestDetails) FailureReason() string {
	return i.Get("failureReason").String()
}
func (i BulkNewDeviceRequestDetails) Line() string {
	return i.Get("line").String()
}
