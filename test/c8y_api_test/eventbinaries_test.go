package api_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/events/eventbinaries"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_CreateEventBinary(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.SetDebug(true)

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "myfile.txt")
	err := os.WriteFile(tempFile, []byte(`foo`), 0644)
	assert.NoError(t, err)

	mo := testcore.CreateManagedObject(t, client)
	assert.NoError(t, mo.Err)

	event := testcore.CreateEvent(t, client, &mo.Data)
	assert.NoError(t, event.Err)
	assert.NotEmpty(t, event.Data.ID())

	binary := client.Events.Binaries.Upsert(context.Background(), event.Data.ID(), eventbinaries.UploadFileOptions{
		FilePath: tempFile,
	})
	assert.NoError(t, binary.Err)
	assert.NotEmpty(t, binary.Data.Name())
}
