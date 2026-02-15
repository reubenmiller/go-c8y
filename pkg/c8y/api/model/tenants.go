package model

import "time"

type ApplicationReferenceCollection struct {
	References []ApplicationReference `json:"references,omitempty"`
	Self       string                 `json:"self,omitempty"`
}

type Tenant struct {
	ID                     string                          `json:"id,omitempty"`
	Name                   string                          `json:"name,omitempty"`
	Self                   string                          `json:"self,omitempty"`
	Status                 string                          `json:"status,omitempty"`
	AdminName              string                          `json:"adminName,omitempty"`
	AdminEmail             string                          `json:"adminEmail,omitempty"`
	AdminPassword          string                          `json:"adminPass,omitempty"`
	Domain                 string                          `json:"domain,omitempty"`
	Company                string                          `json:"company,omitempty"`
	ContactName            string                          `json:"contactName,omitempty"`
	ContactPhone           string                          `json:"contactPhone,omitempty"`
	CustomProperties       map[string]any                  `json:"customProperties,omitempty"`
	Parent                 string                          `json:"parent,omitempty"`
	StorageLimitPerDevice  int64                           `json:"storageLimitPerDevice,omitempty"`
	Applications           *ApplicationReferenceCollection `json:"applications,omitempty"`
	OwnedApplications      *ApplicationReferenceCollection `json:"ownedApplications,omitempty"`
	AllowCreateTenants     bool                            `json:"allowCreateTenants,omitempty"`
	SendPasswordResetEmail bool                            `json:"sendPasswordResetEmail,omitempty"`
}

type TenantCollection struct {
	*BaseResponse

	Tenants []Tenant `json:"tenants,omitempty"`
}

type TenantUsageStatisticsCollection struct {
	*BaseResponse

	UsageStatistics []TenantUsageStatistics `json:"usageStatistics,omitempty"`
}

type TenantUsageStatistics struct {
	Self                              string    `json:"self,omitempty"`
	Day                               time.Time `json:"day,omitzero"`
	DeviceCount                       int64     `json:"deviceCount,omitzero"`
	DeviceWithChildrenCount           int64     `json:"deviceWithChildrenCount,omitzero"`
	DeviceEndpointCount               int64     `json:"deviceEndpointCount,omitzero"`
	DeviceRequestCount                int64     `json:"deviceRequestCount,omitzero"`
	RequestCount                      int64     `json:"requestCount,omitzero"`
	StorageSize                       int64     `json:"storageSize,omitzero"`
	AlarmsCreatedCount                int64     `json:"alarmsCreatedCount,omitzero"`
	AlarmsUpdatedCount                int64     `json:"alarmsUpdatedCount,omitzero"`
	EventsCreatedCount                int64     `json:"eventsCreatedCount,omitzero"`
	EventsUpdatedCount                int64     `json:"eventsUpdatedCount,omitzero"`
	InventoriesCreatedCount           int64     `json:"inventoriesCreatedCount,omitzero"`
	InventoriesUpdatedCount           int64     `json:"inventoriesUpdatedCount,omitzero"`
	MeasurementsCreatedCount          int64     `json:"measurementsCreatedCount,omitzero"`
	OperationsCreatedCount            int64     `json:"operationsCreatedCount,omitzero"`
	OperationsUpdatedCount            int64     `json:"operationsUpdatedCount,omitzero"`
	TotalResourceCreateAndUpdateCount int64     `json:"totalResourceCreateAndUpdateCount,omitzero"`
	SubscribedApplications            []string  `json:"subscribedApplications,omitempty"`

	Resources *TenantUsageResources `json:"resources,omitempty"`
}

type TenantUsageResources struct {
	CPU    int64                        `json:"cpu,omitzero"`
	Memory int64                        `json:"memory,omitzero"`
	UsedBy []TenantUsageResourcesUsedBy `json:"usedBy,omitzero"`
}

type TenantUsageResourcesUsedBy struct {
	CPU    int64  `json:"cpu,omitzero"`
	Memory int64  `json:"memory,omitzero"`
	Name   string `json:"name,omitempty"`
	Cause  string `json:"cause,omitempty"`
}

type TenantUsageStatisticsFile struct {
	ID             string    `json:"id,omitempty"`
	InstanceName   string    `json:"instanceName,omitempty"`
	GenerationDate time.Time `json:"generationDate,omitzero"`
	DateFrom       time.Time `json:"dateFrom,omitzero"`
	DateTo         time.Time `json:"dateTo,omitzero"`
	Type           string    `json:"type,omitempty"`
}

type TenantUsageStatisticsFileCollection struct {
	*BaseResponse

	StatisticsFiles []TenantUsageStatisticsFile `json:"statisticsFiles,omitempty"`
}

type TenantUsageStatisticsSummary struct {
	CreationTime                      time.Time `json:"creationTime,omitzero"`
	TenantDomain                      string    `json:"tenantDomain,omitempty"`
	ParentTenantId                    string    `json:"parentTenantId,omitempty"`
	TenantCompany                     string    `json:"tenantCompany,omitempty"`
	TenantId                          string    `json:"tenantId,omitzero"`
	SubscribedApplications            []string  `json:"subscribedApplications,omitzero"`
	DeviceEndpointCount               int64     `json:"deviceEndpointCount,omitzero"`
	PeakStorageSize                   int64     `json:"peakStorageSize,omitzero"`
	DeviceWithChildrenCount           int64     `json:"deviceWithChildrenCount,omitzero"`
	InventoriesUpdatedCount           int64     `json:"inventoriesUpdatedCount,omitzero"`
	EventsUpdatedCount                int64     `json:"eventsUpdatedCount,omitzero"`
	RequestCount                      int64     `json:"requestCount,omitzero"`
	DeviceCount                       int64     `json:"deviceCount,omitzero"`
	PeakDeviceWithChildrenCount       int64     `json:"peakDeviceWithChildrenCount,omitzero"`
	DeviceRequestCount                int64     `json:"deviceRequestCount,omitzero"`
	StorageLimitPerDevice             int64     `json:"storageLimitPerDevice,omitzero"`
	EventsCreatedCount                int64     `json:"eventsCreatedCount,omitzero"`
	OperationsUpdatedCount            int64     `json:"operationsUpdatedCount,omitzero"`
	AlarmsCreatedCount                int64     `json:"alarmsCreatedCount,omitzero"`
	OperationsCreatedCount            int64     `json:"operationsCreatedCount,omitzero"`
	PeakDeviceCount                   int64     `json:"peakDeviceCount,omitzero"`
	AlarmsUpdatedCount                int64     `json:"alarmsUpdatedCount,omitzero"`
	InventoriesCreatedCount           int64     `json:"inventoriesCreatedCount,omitzero"`
	MeasurementsCreatedCount          int64     `json:"measurementsCreatedCount,omitzero"`
	StorageSize                       int64     `json:"storageSize,omitzero"`
	TotalResourceCreateAndUpdateCount int64     `json:"totalResourceCreateAndUpdateCount,omitzero"`

	TenantCustomProperties map[string]any `json:"tenantCustomProperties,omitzero,omitempty"`

	Resources *TenantUsageResources `json:"resources,omitempty"`
}
