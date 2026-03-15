package api_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/binaries"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_CreateBinary(t *testing.T) {
	client := testcore.CreateTestClient(t)

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "myfile.txt")
	err := os.WriteFile(tempFile, []byte(`foo`), 0644)
	assert.NoError(t, err)
	binary := client.Binaries.Create(context.Background(), binaries.UploadFileOptions{
		FilePath: tempFile,
	})
	t.Cleanup(func() {
		if !binary.IsError() {
			client.Binaries.Delete(context.Background(), binary.Data.ID())
		}
	})
	assert.NoError(t, binary.Err)
	assert.NotEmpty(t, binary.Data.Name())

	resp := client.Binaries.Get(context.Background(), "0")
	assert.Error(t, resp.Err)

	assert.True(t, api.IsNotFound(resp.Err))

	// Get but don't read the response
	binaryFile := client.Binaries.Get(context.Background(), binary.Data.ID())
	assert.NoError(t, err)
	defer binaryFile.Data.Close()
	assert.NotEmpty(t, binaryFile.Data.FileName())
	assert.NotEmpty(t, binaryFile.Data.URL())
	assert.Greater(t, binaryFile.Data.Size(), int64(0))

	var buf bytes.Buffer
	_, err = io.Copy(&buf, binaryFile.Data.Reader())
	assert.NoError(t, err)

	assert.NoError(t, err)
	assert.Equal(t, binaryFile.Data.Response.StatusCode(), 200)
	assert.Equal(t, buf.String(), "foo")
}
