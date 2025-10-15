package c8y_api_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/binaries"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_CreateBinary(t *testing.T) {
	client := testcore.CreateTestClient(t)

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "myfile.txt")
	err := os.WriteFile(tempFile, []byte(`foo`), 0644)
	assert.NoError(t, err)
	binary, err := client.Binaries.Create(context.Background(), binaries.UploadFileOptions{
		FilePath: tempFile,
	})
	t.Cleanup(func() {
		if binary != nil {
			client.Binaries.Delete(context.Background(), binary.ID)
		}
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, binary.Name)

	// TODO: Add special binary handling which will read the response based on the exit code
	resp, err := client.Binaries.Get(context.Background(), "0")
	assert.Error(t, err)
	assert.Nil(t, resp)

	sdkError := err.(*c8y_api.Error)
	assert.Equal(t, sdkError.Code, 404)

	assert.ErrorAs(t, c8y_api.ErrAPINotFound, err)

	// Get but don't read the response
	binaryFile, err := client.Binaries.Get(context.Background(), binary.ID)
	assert.NoError(t, err)
	defer binaryFile.Close()
	assert.NotEmpty(t, binaryFile.FileName())
	assert.NotEmpty(t, binaryFile.URL())
	assert.Greater(t, binaryFile.Size(), int64(0))

	var buf bytes.Buffer
	_, err = io.Copy(&buf, binaryFile.Reader())
	assert.NoError(t, err)

	assert.NoError(t, err)
	assert.Equal(t, binaryFile.Response.StatusCode(), 200)
	assert.Equal(t, buf.String(), "foo")
}
