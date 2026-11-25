// Package core covers types shared by every service: the error types (Error,
// APIError, BatchError), BinaryResponse and NoContent result data, and upload
// options and progress helpers.
//
// Every service returns its errors in Result.Err. Classify them with the
// helpers of the api package.
//
// Example, with a client and ctx in scope:
//
//	result := client.Alarms.Get(ctx, "12345")
//	var apiErr *core.Error
//	if errors.As(result.Err, &apiErr) {
//		fmt.Println(apiErr.Error())
//	}
package core
