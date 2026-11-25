// Package enrollment provides certificate enrollment of devices with a one
// time password and a certificate signing request (EST).
//
// Reach it from a client with client.Devices.Enrollment.
//
// Example, with a client and ctx in scope:
//
//	otp, err := client.Devices.Enrollment.GenerateOneTimePassword()
//	fmt.Println(len(otp) > 0, err)
package enrollment
