package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// DataHubJob is a DataHub (Dremio) query job.
type DataHubJob struct {
	jsondoc.Facade
}

func NewDataHubJob(b []byte) DataHubJob {
	return DataHubJob{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (j DataHubJob) ID() string {
	return j.Get("id").String()
}

func (j DataHubJob) JobState() string {
	return j.Get("jobState").String()
}

// DataHubResult is a single row of a DataHub query result. The columns are
// query-dependent, so it is an untyped document.
type DataHubResult struct {
	jsondoc.Facade
}

func NewDataHubResult(b []byte) DataHubResult {
	return DataHubResult{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// DataHubOffloadingConfiguration is one offloading configuration of the
// DataHub management API (GET /service/datahub/offloadingconfigurations).
type DataHubOffloadingConfiguration struct {
	jsondoc.Facade
}

func NewDataHubOffloadingConfiguration(b []byte) DataHubOffloadingConfiguration {
	return DataHubOffloadingConfiguration{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// DataHubSchedulerJobs is the scheduler's latest-jobs report
// (GET /service/datahub/scheduler/latestjobs), returned as one document.
type DataHubSchedulerJobs struct {
	jsondoc.Facade
}

func NewDataHubSchedulerJobs(b []byte) DataHubSchedulerJobs {
	return DataHubSchedulerJobs{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// DataHubTenant is the DataHub tenant configuration
// (the "data" object of GET /service/datahub/tenant).
type DataHubTenant struct {
	jsondoc.Facade
}

func NewDataHubTenant(b []byte) DataHubTenant {
	return DataHubTenant{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// DataHubUser is one DataHub (Dremio) user
// (GET /service/datahub/dremio/users).
type DataHubUser struct {
	jsondoc.Facade
}

func NewDataHubUser(b []byte) DataHubUser {
	return DataHubUser{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}
