package core

import "errors"

// ErrNameNotFound is matched (errors.Is) by the error of a lookup by name which
// found nothing, for any resource (a device, an application, a firmware, ...).
// Use errors.As with [*NameNotFoundError] to get the resource and the name.
//
// The error is also a 404 [Error], so [IsNotFound] is true for it.
var ErrNameNotFound = errors.New("not found by name")

// NameNotFoundError is the error of a lookup by name which found nothing, for
// example "managed object not found with name: device01". It wraps a 404
// [Error] (which holds the message), so [IsNotFound] and status checks keep
// working, and it matches [ErrNameNotFound].
//
//	var notFound *core.NameNotFoundError
//	if errors.As(err, &notFound) {
//		fmt.Println(notFound.Resource, notFound.Name)
//	}
type NameNotFoundError struct {
	// NotFound is the 404 error, with the message of the lookup.
	NotFound *Error

	// Resource is the kind of resource looked up, e.g. "managed object".
	Resource string

	// Name is the name (or name pattern) which matched nothing.
	Name string
}

// NewNameNotFoundError returns the error of a lookup of the resource by name
// which found nothing. The message is the given format, so the text of an
// existing error does not change.
func NewNameNotFoundError(resource, name, format string, args ...any) *NameNotFoundError {
	return &NameNotFoundError{
		NotFound: ErrNotFound(format, args...),
		Resource: resource,
		Name:     name,
	}
}

func (e *NameNotFoundError) Error() string { return e.NotFound.Error() }

// StatusCode is the status code of the lookup (404).
func (e *NameNotFoundError) StatusCode() int { return e.NotFound.StatusCode() }

// Unwrap gives access to the 404 [Error].
func (e *NameNotFoundError) Unwrap() error { return e.NotFound }

// NotFoundName returns the resource and the name of the failed lookup. See
// [NameNotFound].
func (e *NameNotFoundError) NotFoundName() (resource, name string) { return e.Resource, e.Name }

// Is reports whether target is [ErrNameNotFound].
func (e *NameNotFoundError) Is(target error) bool { return target == ErrNameNotFound }

// LookupError marks an error which happened while a reference (a name, an
// external id, a query) was being resolved to an id, as opposed to an error of
// the request the caller asked for. A request which resolves a name first
// (for example Get with "name:device01") returns the error of the lookup
// wrapped in a LookupError, so a caller can tell the two apart:
//
//	var lookupErr *core.LookupError
//	if errors.As(err, &lookupErr) {
//		// the reference could not be resolved (lookupErr.Resource, lookupErr.Reference)
//	}
//
// The message of the wrapped error is unchanged, and errors.Is/As reach the
// wrapped error (a 404 [Error], a [NameNotFoundError], a transport error, ...).
type LookupError struct {
	// Resource is the kind of resource looked up, e.g. "managed object".
	Resource string

	// Reference is the reference which was resolved, e.g. "name:device01".
	Reference string

	// Err is the error of the lookup.
	Err error
}

func (e *LookupError) Error() string { return e.Err.Error() }

// Unwrap gives access to the error of the lookup.
func (e *LookupError) Unwrap() error { return e.Err }

// WrapLookup marks err as an error of the lookup of a resource. It returns nil
// for a nil error and an error which is already a [LookupError] unchanged.
func WrapLookup(resource, reference string, err error) error {
	if err == nil {
		return nil
	}
	var lookupErr *LookupError
	if errors.As(err, &lookupErr) {
		return err
	}
	return &LookupError{Resource: resource, Reference: reference, Err: err}
}

// IsLookupError reports whether err happened while resolving a reference.
func IsLookupError(err error) bool {
	var lookupErr *LookupError
	return errors.As(err, &lookupErr)
}

type nameNotFound interface {
	error
	NotFoundName() (resource, name string)
}

// NameNotFound reports whether err is (or wraps) the error of a lookup by name
// which found nothing, and returns the kind of resource and the name. It works
// for every such error, including the ones of packages which have an own error
// type, so a caller does not need to know them all:
//
//	if resource, name, ok := core.NameNotFound(err); ok {
//		fmt.Printf("no %s with the name %q\n", resource, name)
//	}
func NameNotFound(err error) (resource, name string, ok bool) {
	var notFound nameNotFound
	if !errors.As(err, &notFound) {
		return "", "", false
	}
	resource, name = notFound.NotFoundName()
	return resource, name, true
}
