package c8y_api

import (
	"context"
	"log/slog"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alarms"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
)

// Create a Cumulocity client and use it to query the platform
func Example_newClient() {
	client := NewClientFromEnvironment(ClientOptions{})

	alarmCollection := client.Alarms.List(context.Background(), alarms.ListOptions{
		Severity: []string{
			model.AlarmSeverityMajor,
		},
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 100,
		},
	})
	if alarmCollection.Err != nil {
		panic(alarmCollection.Err)
	}
	for alarm := range jsondoc.DecodeIter[model.Alarm](alarmCollection.Data.Iter()) {
		slog.Info("alarm", "id", alarm.ID, "text", alarm.Text)
	}
}
