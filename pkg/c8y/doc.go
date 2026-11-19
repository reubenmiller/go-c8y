// Package c8y holds transport-level helpers that sit below the API client:
// a file-backed HTTP response cache ([NewCachedTransport]) and small value
// helpers such as [IsID].
//
// The Cumulocity API client itself lives in package
// github.com/reubenmiller/go-c8y/v2/pkg/c8y/api. Everything exported here is
// supported v2 API and follows semantic versioning.
package c8y
