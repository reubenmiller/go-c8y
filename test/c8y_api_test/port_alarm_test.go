package api_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func TestAlarmService_CreateAlarm(t *testing.T) {
	client := testcore.CreateTestClient(t)
	testDevice := testcore.CreateDevice(t, client)

	value := model.Alarm{
		Type:     "testAlarm",
		Text:     "Test alarm",
		Severity: "MAJOR",
		Time:     time.Now(),
		Source:   model.NewSource(testDevice.Data.ID()),
	}

	alarm := client.Alarms.Create(context.Background(), value)
	assert.NoError(t, alarm.Err)
	assert.Equal(t, http.StatusCreated, alarm.HTTPStatus)
	assert.NotEmpty(t, alarm.Data.Length())
}

func TestAlarmService_UpdateAlarm(t *testing.T) {
	client := testcore.CreateTestClient(t)
	testDevice := testcore.CreateDevice(t, client)

	alarm := client.Alarms.Create(
		context.Background(),
		model.Alarm{
			Time:     time.Now(),
			Source:   model.NewSource(testDevice.Data.ID()),
			Severity: "MAJOR",
			Text:     "Test Alarm 1",
			Type:     "TestAlarm1",
		},
	)
	assert.NoError(t, alarm.Err)
	assert.Equal(t, http.StatusCreated, alarm.HTTPStatus)
	assert.NotEmpty(t, alarm.Data.Length())

	// Update severity
	updatedAlarm1 := client.Alarms.Update(
		context.Background(),
		alarm.Data.ID(),
		model.AlarmUpdateProperties{
			Severity: "CRITICAL",
		})

	assert.NoError(t, updatedAlarm1.Err)
	assert.Equal(t, http.StatusCreated, alarm.HTTPStatus)
	assert.Equal(t, "CRITICAL", updatedAlarm1.Data.Severity())

	// Update Text
	updatedAlarm1 = client.Alarms.Update(
		context.Background(),
		alarm.Data.ID(),
		model.AlarmUpdateProperties{
			Text: "Updated Alarm Text 1",
		})

	assert.NoError(t, updatedAlarm1.Err)
	assert.Equal(t, http.StatusOK, updatedAlarm1.HTTPStatus)
	testingutils.Equals(t, "Updated Alarm Text 1", updatedAlarm1.Data.Text())

	// Update Status
	updatedAlarm1 = client.Alarms.Update(
		context.Background(),
		alarm.Data.ID(),
		model.AlarmUpdateProperties{
			Status: "ACKNOWLEDGED",
		})

	assert.NoError(t, updatedAlarm1.Err)
	assert.Equal(t, http.StatusOK, updatedAlarm1.HTTPStatus)
	testingutils.Equals(t, "ACKNOWLEDGED", updatedAlarm1.Data.Status())

	// Update all fields at once
	updatedAlarm1 = client.Alarms.Update(
		context.Background(),
		alarm.Data.ID(),
		model.AlarmUpdateProperties{
			Status:   "CLEARED",
			Text:     "Alarm is cleared",
			Severity: "MINOR",
		})

	assert.NoError(t, updatedAlarm1.Err)
	assert.Equal(t, http.StatusOK, updatedAlarm1.HTTPStatus)
	testingutils.Equals(t, "CLEARED", updatedAlarm1.Data.Status())
	testingutils.Equals(t, "MINOR", updatedAlarm1.Data.Severity())
	testingutils.Equals(t, "Alarm is cleared", updatedAlarm1.Data.Text())
}

func TestAlarmService_GetAlarmByID(t *testing.T) {
	client := testcore.CreateTestClient(t)
	testDevice := testcore.CreateDevice(t, client)

	alarm := client.Alarms.Create(context.Background(), model.Alarm{
		Time:     time.Now(),
		Source:   model.NewSource(testDevice.Data.ID()),
		Severity: "MAJOR",
		Text:     "Test Alarm 1",
		Type:     "TestAlarm1",
	})
	assert.NoError(t, alarm.Err)
	assert.Equal(t, http.StatusCreated, alarm.HTTPStatus)
	assert.NotEmpty(t, alarm.Data)

	alarm2 := client.Alarms.Get(context.Background(), alarm.Data.ID())
	assert.NoError(t, alarm2.Err)
	assert.Equal(t, http.StatusOK, alarm2.HTTPStatus)
	assert.Equal(t, alarm.Data.ID(), alarm2.Data.ID())
}

func TestAlarmService_GetAlarmCollection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	testDevice := testcore.CreateDevice(t, client)

	alarmFactory := func(alarmType string) op.Result[jsonmodels.Alarm] {
		alarm := model.Alarm{
			Time:     time.Now(),
			Source:   model.NewSource(testDevice.Data.ID()),
			Severity: "MAJOR",
			Text:     "Test Alarm 1",
			Type:     alarmType,
		}

		alarmObj := client.Alarms.Create(context.Background(), alarm)
		assert.NoError(t, alarmObj.Err)
		assert.Equal(t, http.StatusCreated, alarmObj.HTTPStatus)
		assert.NotEmpty(t, alarmObj.Data)
		time.Sleep(1000 * time.Millisecond)
		return alarmObj
	}

	alarm1 := alarmFactory("alarm1")
	alarm2 := alarmFactory("alarm2")
	alarm3 := alarmFactory("alarm3")

	// Filter by Source and Severity
	alarmCollection := client.Alarms.List(
		context.Background(),
		alarms.ListOptions{
			Source:   testDevice.Data.ID(),
			Severity: []string{"MAJOR"},
			DateTo:   time.Now(),
		},
	)
	assert.NoError(t, alarmCollection.Err)
	assert.Equal(t, http.StatusOK, alarmCollection.HTTPStatus)
	assert.Equal(t, 3, alarmCollection.Data.Length())

	// Convert Result to slice for easier testing
	alarmList, err := op.ToSliceR(alarmCollection)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(alarmList))

	// alarms will be in reverse order due to dateFrom filtering
	assert.Equal(t, alarm3.Data.ID(), alarmList[0].ID())
	assert.Equal(t, alarm2.Data.ID(), alarmList[1].ID())
	assert.Equal(t, alarm1.Data.ID(), alarmList[2].ID())

	// Filter by Source and Type
	alarmCollection = client.Alarms.List(
		context.Background(),
		alarms.ListOptions{
			Source: testDevice.Data.ID(),
			Type:   []string{"alarm2"},
		},
	)
	assert.NoError(t, alarmCollection.Err)
	assert.Equal(t, http.StatusOK, alarmCollection.HTTPStatus)
	assert.Equal(t, 1, alarmCollection.Data.Length())

	alarmList2, err := op.ToSliceR(alarmCollection)
	assert.NoError(t, err)
	assert.Equal(t, alarm2.Data.ID(), alarmList2[0].ID())
}

func TestAlarmService_BulkUpdateAlarms(t *testing.T) {
	client := testcore.CreateTestClient(t)
	testDevice := testcore.CreateDevice(t, client)

	alarmFactory := func(alarmType string) op.Result[jsonmodels.Alarm] {
		alarm := model.Alarm{
			Time:     time.Now(),
			Source:   model.NewSource(testDevice.Data.ID()),
			Severity: "MAJOR",
			Text:     "Test Alarm 1",
			Type:     alarmType,
		}

		alarmObj := client.Alarms.Create(context.Background(), alarm)
		assert.NoError(t, alarmObj.Err)
		assert.Equal(t, http.StatusCreated, alarmObj.HTTPStatus)
		assert.NotEmpty(t, alarmObj.Data)
		time.Sleep(1000 * time.Millisecond)
		return alarmObj
	}

	alarm1 := alarmFactory("alarm1")
	alarm2 := alarmFactory("alarm2")
	alarm3 := alarmFactory("alarm3")

	client.Client.SetDebug(true)
	// Update list of alarms
	updatedList := client.Alarms.UpdateList(
		context.Background(),
		alarms.BulkUpdateOptions{
			Source: testDevice.Data.ID(),
			Status: []string{"ACTIVE"},
		},
		model.Alarm{
			Status: "CLEARED",
		},
	)

	/*
		Note:
		If the StatusCode is "Accepted", then the error will be set to
		"job scheduled on Cumulocity side; try again later" even though the request was accepted.
		Also, a delay is required before getting the status in the platform.

		Reference: https://cumulocity.com/guides/reference/alarms/
		"Since this operations can take a lot of time, request returns after maximum 0.5 sec of processing, and updating is continued as a background process in the platform."
	*/
	if updatedList.HTTPStatus == http.StatusAccepted {
		// Wait for Cumulocity to process the request in the background
		time.Sleep(5 * time.Second)
	}

	assert.NoError(t, updatedList.Err)
	assert.True(t, updatedList.HTTPStatus == http.StatusAccepted || updatedList.HTTPStatus == http.StatusOK, "Accepted or OK")

	// Filter by Source and Severity
	alarmCollection := client.Alarms.List(
		context.Background(),
		alarms.ListOptions{
			Source:   testDevice.Data.ID(),
			Status:   []string{"CLEARED"},
			DateFrom: time.Now().Add(-60 * time.Second),
			DateTo:   time.Now(),
		},
	)
	assert.NoError(t, alarmCollection.Err)
	assert.Equal(t, http.StatusOK, alarmCollection.HTTPStatus)
	assert.Equal(t, 3, alarmCollection.Data.Length())

	alarmList, err := op.ToSliceR(alarmCollection)
	assert.NoError(t, err)

	// should be in reverse order
	testingutils.Equals(t, alarm1.Data.ID(), alarmList[2].ID())
	testingutils.Equals(t, alarm2.Data.ID(), alarmList[1].ID())
	testingutils.Equals(t, alarm3.Data.ID(), alarmList[0].ID())
}

func TestAlarmService_RemoveAlarmCollection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	testDevice := testcore.CreateDevice(t, client)

	alarmFactory := func(alarmType string) op.Result[jsonmodels.Alarm] {
		alarm := model.Alarm{
			Time:     time.Now(),
			Source:   model.NewSource(testDevice.Data.ID()),
			Severity: "MAJOR",
			Text:     "Test Alarm 1",
			Type:     alarmType,
		}

		alarmObj := client.Alarms.Create(context.Background(), alarm)
		assert.NoError(t, alarmObj.Err)
		assert.Equal(t, http.StatusCreated, alarmObj.HTTPStatus)
		assert.NotEmpty(t, alarmObj.Data)
		time.Sleep(1000 * time.Millisecond)
		return alarmObj
	}

	alarmFactory("customAlarm1")
	alarmFactory("customAlarm2")
	alarmFactory("customAlarm3")

	// Get alarms before deletion
	alarmCollection := client.Alarms.List(
		context.Background(),
		alarms.ListOptions{
			Source: testDevice.Data.ID(),
		},
	)

	assert.NoError(t, alarmCollection.Err)
	testingutils.Equals(t, 3, alarmCollection.Data.Length())

	// Delete alarms
	deleteList := client.Alarms.DeleteList(
		context.Background(),
		alarms.DeleteListOptions{
			Source: testDevice.Data.ID(),
		},
	)

	assert.NoError(t, deleteList.Err)
	assert.Equal(t, http.StatusNoContent, deleteList.HTTPStatus)

	// Give server some time to delete
	time.Sleep(1 * time.Second)

	// Get alarms after deletion
	alarmCollection = client.Alarms.List(
		context.Background(),
		alarms.ListOptions{
			Source: testDevice.Data.ID(),
		},
	)

	assert.NoError(t, alarmCollection.Err)
	testingutils.Equals(t, 0, alarmCollection.Data.Length())
}
