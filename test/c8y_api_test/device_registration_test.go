package api_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/users"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_BulkRegistration(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.SetDebug(true)

	registrations := []model.BulkRegistrationRecord{
		{
			ID:       testingutils.RandomString(32),
			AuthType: model.BulkRegistrationAuthTypeCertificates,
			IsAgent:  true,
		},
		func(m *model.BulkRegistrationRecord) model.BulkRegistrationRecord {
			m.Name = fmt.Sprintf("Custom name %s", m.ID)
			return *m
		}(&model.BulkRegistrationRecord{
			ID:          testingutils.RandomString(32),
			Credentials: testingutils.RandomPassword(32),
			AuthType:    model.BulkRegistrationAuthTypeBasic,
			IsAgent:     true,
		}),
	}

	t.Cleanup(func() {
		for _, device := range registrations {
			client.Users.Delete(context.Background(), users.DeleteOptions{
				ID: users.ByDeviceUser(device.ID),
			})
		}
	})

	b := &bytes.Buffer{}
	err := model.BulkRegistrationRecordWriter(b, registrations...)
	assert.NoError(t, err)
	result := client.Devices.Registration.CreateBulk(context.Background(), core.UploadFileOptions{
		Reader: b,
	})
	assert.NoError(t, result.Err)
	assert.Equal(t, int64(2), result.Data.TotalSuccessful())
	assert.Equal(t, int64(2), result.Data.TotalCreated())
	assert.Equal(t, int64(2), result.Data.Total())
	assert.Equal(t, int64(0), result.Data.TotalFailed())
}
