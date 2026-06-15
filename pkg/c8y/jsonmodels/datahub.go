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
