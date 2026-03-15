package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/tidwall/gjson"
)

// Notification2Subscription notification subscription object
type Notification2Subscription struct {
	jsondoc.Facade
}

func NewNotification2Subscription(data []byte) Notification2Subscription {
	return Notification2Subscription{jsondoc.Facade{JSONDoc: jsondoc.New(data)}}
}

func (n Notification2Subscription) ID() string {
	return n.Get("id").String()
}

func (n Notification2Subscription) Self() string {
	return n.Get("self").String()
}

func (n Notification2Subscription) Context() string {
	return n.Get("context").String()
}

func (n Notification2Subscription) Subscription() string {
	return n.Get("subscription").String()
}

func (n Notification2Subscription) SourceID() string {
	return n.Get("source.id").String()
}

func (n Notification2Subscription) SourceSelf() string {
	return n.Get("source.self").String()
}

func (n Notification2Subscription) FragmentsToCopy() []string {
	fragments := make([]string, 0)
	n.Get("fragmentsToCopy").ForEach(func(key, value gjson.Result) bool {
		fragments = append(fragments, value.String())
		return true
	})
	return fragments
}

func (n Notification2Subscription) SubscriptionFilterApis() []string {
	apis := make([]string, 0)
	n.Get("subscriptionFilter.apis").ForEach(func(key, value gjson.Result) bool {
		apis = append(apis, value.String())
		return true
	})
	return apis
}

func (n Notification2Subscription) SubscriptionFilterTypeFilter() string {
	return n.Get("subscriptionFilter.typeFilter").String()
}
