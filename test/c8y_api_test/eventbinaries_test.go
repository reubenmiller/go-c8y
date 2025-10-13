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
	// TODO: Create mo and event to be used in the test
	binary, err := client.Events.Binaries.Upsert(context.Background(), "747256", eventbinaries.UploadFileOptions{
		Filename: tempFile,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, binary.Name)
}
