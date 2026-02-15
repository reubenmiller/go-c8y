package jsonmodels

import (
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
)

type UsageStatisticsFile struct {
	jsondoc.Facade
}

func NewUsageStatisticsFile(b []byte) UsageStatisticsFile {
	return UsageStatisticsFile{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (u UsageStatisticsFile) ID() string {
	return u.Get("id").String()
}

func (u UsageStatisticsFile) InstanceName() string {
	return u.Get("instanceName").String()
}

func (u UsageStatisticsFile) GenerationDate() time.Time {
	return u.Get("generationDate").Time()
}

func (u UsageStatisticsFile) DateFrom() time.Time {
	return u.Get("dateFrom").Time()
}

func (u UsageStatisticsFile) DateTo() time.Time {
	return u.Get("dateTo").Time()
}

func (u UsageStatisticsFile) Type() string {
	return u.Get("type").String()
}
