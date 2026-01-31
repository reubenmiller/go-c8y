package c8y_api_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/events/eventbinaries"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_CreateEventBinary(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "myfile.txt")
	err := os.WriteFile(tempFile, []byte(`foo`), 0644)
	assert.NoError(t, err)

	mo := testcore.CreateManagedObject(t, client)
	assert.NoError(t, mo.Err)

	event := testcore.CreateEvent(t, client, &mo.Data)
	assert.NoError(t, event.Err)
	assert.NotEmpty(t, event.Data.ID())

	binary, err := client.Events.Binaries.Upsert(context.Background(), event.Data.ID(), eventbinaries.UploadFileOptions{
		FilePath: tempFile,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, binary.Name)
}
