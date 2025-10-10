package c8y_api

import "resty.dev/v3"

var HeaderProcessingMode = "X-Cumulocity-Processing-Mode"

var ProcessingModePersistent = "PERSISTENT"
var ProcessingModeTransient = "TRANSIENT"
var ProcessingModeQuiescent = "QUIESCENT"
var ProcessingModeCEP = "CEP"

func SetProcessingMode(mode string) resty.RequestFunc {
	return func(r *resty.Request) *resty.Request {
		r.SetHeader(HeaderProcessingMode, mode)
		return r
	}
}

func SetProcessingModePersistent() resty.RequestFunc {
	return SetProcessingMode(ProcessingModePersistent)
}

func SetProcessingModeTransient() resty.RequestFunc {
	return SetProcessingMode(ProcessingModeTransient)
}

func SetProcessingModeQuiescent() resty.RequestFunc {
	return SetProcessingMode(ProcessingModeQuiescent)
}

func SetProcessingModeCEP() resty.RequestFunc {
	return SetProcessingMode(ProcessingModeCEP)
}
