package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/repository/software/softwareversions"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_SoftwareVersion(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	// Get Or Create software item (new item)
	softwareName := "software" + testingutils.RandomString(16)
	version := "1.0.1"
	softwareType := "ci-artifact"
	softwareVersion, found, err := client.Repository.Software.Versions.GetOrCreate(context.Background(), softwareversions.GetOrCreateOptions{
		Software: model.Software{
			Name:         softwareName,
			Type:         "c8y_Software",
			SoftwareType: softwareType,
			Description:  "Some custom artifact name",
		},
		Version: model.SoftwareVersion{
			Version: version,
			// URL:     "dummy",
		},
		File: softwareversions.UploadFileOptions{
			FilePath: "dummy",
		},
	})
	assert.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, version, softwareVersion.C8Y_Software.Version)
	assert.NotEmpty(t, softwareVersion.ID)

	// get
	softwareVersion2, err := client.Repository.Software.Versions.Get(context.Background(), softwareversions.GetOptions{
		ID:          softwareVersion.ID,
		WithParents: true,
	})
	assert.NoError(t, err)
	assert.Equal(t, softwareVersion.ID, softwareVersion2.ID)

	t.Cleanup(func() {
		if softwareVersion2.AdditionParents != nil && len(softwareVersion2.AdditionParents.References) > 0 {
			client.ManagedObjects.Delete(t.Context(), softwareVersion2.AdditionParents.References[0].ManagedObject.ID, managedobjects.DeleteOptions{
				ForceCascade: true,
			})
		}

		// client.ManagedObjects.Delete(t.Context(), softwareVersion.ID, managedobjects.DeleteOptions{
		// 	ForceCascade: true,
		// })
	})

}
