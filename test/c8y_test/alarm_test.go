package c8y_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func TestAlarmService_CreateAlarm(t *testing.T) {
	client := createTestClient()

	testDevice, _, err := client.Inventory.CreateDevice(context.Background(), "testDevice")
	testingutils.Ok(t, err)
	defer client.Inventory.Delete(context.Background(), testDevice.ID)

	value := c8y.Alarm{
		Type:     "testalarm",
		Text:     "Test alarm",
		Severity: "MAJOR",
		Time:     c8y.NewTimestamp(),
		Source:   c8y.NewSource(testDevice.ID),
	}

	alarm, resp, err := client.Alarm.Create(context.Background(), value)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode)
	testingutils.Assert(t, alarm != nil, "Alarm object should not be empty")
}

func TestAlarmService_UpdateAlarm(t *testing.T) {
	client := createTestClient()

	testDevice, _, err := client.Inventory.CreateDevice(context.Background(), "testDevice")
	testingutils.Ok(t, err)
	defer client.Inventory.Delete(context.Background(), testDevice.ID)

	alarm, resp, err := client.Alarm.Create(
		context.Background(),
		c8y.Alarm{
			Time:     c8y.NewTimestamp(),
			Source:   c8y.NewSource(testDevice.ID),
			Severity: "MAJOR",
			Text:     "Test Alarm 1",
			Type:     "TestAlarm1",
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode)
	testingutils.Assert(t, alarm != nil, "Alarm should not be nil", alarm)

	// Update serverity
	updatedAlarm1, resp, err := client.Alarm.Update(
		context.Background(),
		alarm.ID,
		c8y.AlarmUpdateProperties{
			Severity: "CRITICAL",
		})

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, "CRITICAL", updatedAlarm1.Severity)

	// Update Text
	updatedAlarm1, resp, err = client.Alarm.Update(
		context.Background(),
		alarm.ID,
		c8y.AlarmUpdateProperties{
			Text: "Updated Alarm Text 1",
		})

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, "Updated Alarm Text 1", updatedAlarm1.Text)

	// Update Status
	updatedAlarm1, resp, err = client.Alarm.Update(
		context.Background(),
		alarm.ID,
		c8y.AlarmUpdateProperties{
			Status: "ACKNOWLEDGED",
		})

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, "ACKNOWLEDGED", updatedAlarm1.Status)

	// Update all fields at once
	updatedAlarm1, resp, err = client.Alarm.Update(
		context.Background(),
		alarm.ID,
		c8y.AlarmUpdateProperties{
			Status:   "CLEARED",
			Text:     "Alarm is cleared",
			Severity: "MINOR",
		})

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, "CLEARED", updatedAlarm1.Status)
	testingutils.Equals(t, "MINOR", updatedAlarm1.Severity)
	testingutils.Equals(t, "Alarm is cleared", updatedAlarm1.Text)
}

func TestAlarmService_GetAlarmByID(t *testing.T) {
	client := createTestClient()

	testDevice, _, err := client.Inventory.CreateDevice(context.Background(), "testDevice")
	testingutils.Ok(t, err)
	defer client.Inventory.Delete(context.Background(), testDevice.ID)

	alarm, resp, err := client.Alarm.Create(context.Background(), c8y.Alarm{
		Time:     c8y.NewTimestamp(),
		Source:   c8y.NewSource(testDevice.ID),
		Severity: "MAJOR",
		Text:     "Test Alarm 1",
		Type:     "TestAlarm1",
	})
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode)
	testingutils.Assert(t, alarm != nil, "Alarm should not be nil", alarm)

	alarm2, resp, err := client.Alarm.GetAlarm(context.Background(), alarm.ID)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, alarm.ID, alarm2.ID)
}

func TestAlarmService_GetAlarmCollection(t *testing.T) {
	client := createTestClient()

	testDevice, _, err := client.Inventory.CreateDevice(context.Background(), "testDevice")
	testingutils.Ok(t, err)
	defer client.Inventory.Delete(context.Background(), testDevice.ID)

	alarmFactory := func(alarmtype string) *c8y.Alarm {
		alarm := c8y.Alarm{
			Time:     c8y.NewTimestamp(),
			Source:   c8y.NewSource(testDevice.ID),
			Severity: "MAJOR",
			Text:     "Test Alarm 1",
			Type:     alarmtype,
		}

		alarmObj, resp, err := client.Alarm.Create(context.Background(), alarm)
		testingutils.Ok(t, err)
		testingutils.Equals(t, http.StatusCreated, resp.StatusCode)
		testingutils.Assert(t, alarmObj != nil, "Alarm should not be nil", alarmObj)
		return alarmObj
	}

	alarm1 := alarmFactory("alarm1")
	alarm2 := alarmFactory("alarm2")
	alarm3 := alarmFactory("alarm3")

	// Filter by Source and Severity
	alarmCollection, resp, err := client.Alarm.GetAlarms(
		context.Background(),
		&c8y.AlarmCollectionOptions{
			Source:   testDevice.ID,
			Severity: "MAJOR",
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, 3, len(alarmCollection.Alarms))
	testingutils.Equals(t, 3, len(alarmCollection.Items))
	testingutils.Equals(t, alarm1.ID, alarmCollection.Alarms[0].ID)
	testingutils.Equals(t, alarm2.ID, alarmCollection.Alarms[1].ID)
	testingutils.Equals(t, alarm3.ID, alarmCollection.Alarms[2].ID)

	// Filter by Source and Type
	alarmCollection, resp, err = client.Alarm.GetAlarms(
		context.Background(),
		&c8y.AlarmCollectionOptions{
			Source: testDevice.ID,
			Type:   "alarm2",
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, 1, len(alarmCollection.Alarms))
	testingutils.Equals(t, alarm2.ID, alarmCollection.Alarms[0].ID)
}

func TestAlarmService_BulkUpdateAlarms(t *testing.T) {
	client := createTestClient()

	testDevice, _, err := client.Inventory.CreateDevice(context.Background(), "testDevice")
	testingutils.Ok(t, err)
	defer client.Inventory.Delete(context.Background(), testDevice.ID)

	alarmFactory := func(alarmtype string) *c8y.Alarm {
		alarm := c8y.Alarm{
			Time:     c8y.NewTimestamp(),
			Source:   c8y.NewSource(testDevice.ID),
			Severity: "MAJOR",
			Text:     "Test Alarm 1",
			Type:     alarmtype,
		}

		alarmObj, resp, err := client.Alarm.Create(context.Background(), alarm)
		testingutils.Ok(t, err)
		testingutils.Equals(t, http.StatusCreated, resp.StatusCode)
		testingutils.Assert(t, alarmObj != nil, "Alarm should not be nil", alarmObj)
		return alarmObj
	}

	alarm1 := alarmFactory("alarm1")
	alarm2 := alarmFactory("alarm2")
	alarm3 := alarmFactory("alarm3")

	resp, err := client.Alarm.BulkUpdateAlarms(
		context.Background(),
		"CLEARED",
		c8y.AlarmUpdateOptions{
			Source: testDevice.ID,
			Status: "ACTIVE",
		},
	)

	/*
		Note:
		If the StatusCode is "Accepted", then the error will be set to
		"job scheduled on Cumulocity side; try again later" eventhough the request was accepted.
		Also, a delay is required before getting the status in the platform.

		Reference: https://cumulocity.com/guides/reference/alarms/
		"Since this operations can take a lot of time, request returns after maximum 0.5 sec of processing, and updating is continued as a background process in the platform."
	*/
	testingutils.Assert(t, resp != nil, "Response should not be nil")

	switch resp.StatusCode {
	case http.StatusAccepted:
		testingutils.Equals(t, c8y.AcceptedError{}, err)
		// Wait for Cumulocity to process the request in the background
		time.Sleep(5 * time.Second)

	case http.StatusOK:
		testingutils.Ok(t, err)

	default:
		t.Error("Unexpected error code. Expected either StatusAccepted (202) or StatusOK (200)")
	}

	testingutils.Ok(t, err)
	testingutils.Assert(t, resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusOK, "Accepted or OK")

	// Filter by Source and Severity
	alarmCollection, resp, err := client.Alarm.GetAlarms(
		context.Background(),
		&c8y.AlarmCollectionOptions{
			Source: testDevice.ID,
			Status: "CLEARED",
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, 3, len(alarmCollection.Alarms))
	testingutils.Equals(t, alarm1.ID, alarmCollection.Alarms[0].ID)
	testingutils.Equals(t, alarm2.ID, alarmCollection.Alarms[1].ID)
	testingutils.Equals(t, alarm3.ID, alarmCollection.Alarms[2].ID)

}

func TestAlarmService_RemoveAlarmCollection(t *testing.T) {
	client := createTestClient()

	testDevice, _, err := client.Inventory.CreateDevice(
		context.Background(),
		"testDevice",
	)
	testingutils.Ok(t, err)
	defer client.Inventory.Delete(context.Background(), testDevice.ID)

	alarmFactory := func(alarmtype string) *c8y.Alarm {
		alarm := c8y.Alarm{
			Time:     c8y.NewTimestamp(),
			Source:   c8y.NewSource(testDevice.ID),
			Severity: "MAJOR",
			Text:     "Test Alarm 1",
			Type:     alarmtype,
		}

		alarmObj, resp, err := client.Alarm.Create(
			context.Background(),
			alarm,
		)
		testingutils.Ok(t, err)
		testingutils.Equals(t, http.StatusCreated, resp.StatusCode)
		testingutils.Assert(t, alarmObj != nil, "Alarm should not be nil", alarmObj)
		return alarmObj
	}

	alarmFactory("customAlarm1")
	alarmFactory("customAlarm2")
	alarmFactory("customAlarm3")

	// Get alarms before deletion
	alarmCollection, resp, err := client.Alarm.GetAlarms(
		context.Background(),
		&c8y.AlarmCollectionOptions{
			Source: testDevice.ID,
		},
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, 3, len(alarmCollection.Alarms))

	// Delete alarms
	resp, err = client.Alarm.DeleteAlarms(
		context.Background(),
		&c8y.AlarmCollectionOptions{
			Source: testDevice.ID,
		})

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode)

	// Get alarms after deletion
	alarmCollection, resp, err = client.Alarm.GetAlarms(
		context.Background(),
		&c8y.AlarmCollectionOptions{
			Source: testDevice.ID,
		},
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, 0, len(alarmCollection.Alarms))
}
