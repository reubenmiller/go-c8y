package jsonmodels

import (
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

type TenantUsageStatistics struct {
	jsondoc.JSONDoc
}

func NewTenantUsageStatistics(b []byte) TenantUsageStatistics {
	return TenantUsageStatistics{jsondoc.New(b)}
}

// Self returns the URL to this resource
func (s TenantUsageStatistics) Self() string {
	return s.Get("self").String()
}

// Day returns the day for which the statistics are calculated
func (s TenantUsageStatistics) Day() time.Time {
	return s.Get("day").Time()
}

// DeviceCount returns the count of devices
func (s TenantUsageStatistics) DeviceCount() int64 {
	return s.Get("deviceCount").Int()
}

// RequestCount returns the total count of requests
func (s TenantUsageStatistics) RequestCount() int64 {
	return s.Get("requestCount").Int()
}

// StorageSize returns the storage size in bytes
func (s TenantUsageStatistics) StorageSize() int64 {
	return s.Get("storageSize").Int()
}

type TenantUsageStatisticsSummary struct {
	jsondoc.JSONDoc
}

func NewTenantUsageStatisticsSummary(b []byte) TenantUsageStatisticsSummary {
	return TenantUsageStatisticsSummary{jsondoc.New(b)}
}

// TenantId returns the tenant ID
func (s TenantUsageStatisticsSummary) TenantId() string {
	return s.Get("tenantId").String()
}

// TenantDomain returns the tenant domain
func (s TenantUsageStatisticsSummary) TenantDomain() string {
	return s.Get("tenantDomain").String()
}

// TenantCompany returns the tenant company name
func (s TenantUsageStatisticsSummary) TenantCompany() string {
	return s.Get("tenantCompany").String()
}

// CreationTime returns the creation time
func (s TenantUsageStatisticsSummary) CreationTime() time.Time {
	return s.Get("creationTime").Time()
}

// DeviceCount returns the count of devices
func (s TenantUsageStatisticsSummary) DeviceCount() int64 {
	return s.Get("deviceCount").Int()
}

// StorageSize returns the storage size in bytes
func (s TenantUsageStatisticsSummary) StorageSize() int64 {
	return s.Get("storageSize").Int()
}
