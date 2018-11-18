package c8y

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func createClient() *Client {
	host := os.Getenv("C8Y_HOST")
	tenant := os.Getenv("C8Y_TENANT")
	username := os.Getenv("C8Y_USERNAME")
	password := os.Getenv("C8Y_PASSWORD")

	fmt.Printf("Host %s, Tenant %s, Username %s, Password %s", host, tenant, username, password)
	client := NewClient(nil, host, tenant, username, password)
	return client
}

func TestMeasurementService_GetMeasurementCollection(t *testing.T) {
	client := createClient()
	dateFrom, dateTo := GetDateRange("10min")
	_, resp, _ := client.Measurement.GetMeasurementCollection(context.Background(), &MeasurementCollectionOptions{
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Source:   "",
	})
	value := resp.JSON.Get("nx_WEA_27_Delta").String()

	fmt.Printf("JSON value: %s", value)
}

/* func TestMeasurementService_GetMeasurementCollection(t *testing.T) {
	type args struct {
		ctx context.Context
		opt *MeasurementCollectionOptions
	}
	tests := []struct {
		name    string
		s       *MeasurementService
		args    args
		want    *MeasurementCollection
		want1   *Response
		wantErr bool
	}{
		// TODO: Add test cases.
		{name: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := tt.s.GetMeasurementCollection(tt.args.ctx, tt.args.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("MeasurementService.GetMeasurementCollection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MeasurementService.GetMeasurementCollection() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("MeasurementService.GetMeasurementCollection() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
} */
