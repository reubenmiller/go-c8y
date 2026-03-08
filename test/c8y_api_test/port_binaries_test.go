package api_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/binaries"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

func TestInventoryService_CreateUpdateDeleteBinary(t *testing.T) {
	client := testcore.CreateTestClient(t)

	testfile1 := testcore.NewDummyFile(t, "testfile1.txt", "test contents 1")
	testfile2 := testcore.NewDummyFile(t, "testfile2.txt", "test contents 2")

	// Upload a new binary
	binary1 := client.Binaries.Create(context.Background(), binaries.UploadFileOptions{
		FilePath: testfile1,
	})
	assert.NoError(t, binary1.Err)
	assert.NotEmpty(t, binary1.Data.ID(), "Binary ID should not be an empty string")
	assert.Equal(t, 201, binary1.HTTPStatus)

	// Cleanup binary when test completes
	t.Cleanup(func() {
		client.Binaries.Delete(context.Background(), binary1.Data.ID())
	})

	// Download the binary, and check if it matches the file that was uploaded exactly
	downloadedBinary1 := client.Binaries.Get(context.Background(), binary1.Data.ID())
	require.NoError(t, downloadedBinary1.Err)
	defer downloadedBinary1.Data.Close()

	var buf1 bytes.Buffer
	_, err := io.Copy(&buf1, downloadedBinary1.Data.Reader())
	assert.NoError(t, err)

	testfile1Contents, err := os.ReadFile(testfile1)
	assert.NoError(t, err)
	assert.Equal(t, string(testfile1Contents), buf1.String())

	// Update the existing binary with a new binary
	binary2 := client.Binaries.Update(context.Background(), binary1.Data.ID(), binaries.UploadFileOptions{
		FilePath: testfile2,
	})
	assert.NoError(t, binary2.Err)
	assert.NotEmpty(t, binary2.Data.ID(), "Binary id should not be an empty string")

	// Download the updated binary and check if it matches the new binary contents
	downloadedBinary2 := client.Binaries.Get(context.Background(), binary2.Data.ID())
	assert.NoError(t, downloadedBinary2.Err)
	if downloadedBinary2.IsError() {
		return // Skip rest of test if download failed
	}
	defer downloadedBinary2.Data.Close()

	var buf2 bytes.Buffer
	_, err = io.Copy(&buf2, downloadedBinary2.Data.Reader())
	assert.NoError(t, err)

	testfile2Contents, err := os.ReadFile(testfile2)
	assert.NoError(t, err)
	assert.Equal(t, string(testfile2Contents), buf2.String())

	// Delete the binary
	deleteResp := client.Binaries.Delete(context.Background(), binary2.Data.ID())
	assert.NoError(t, deleteResp.Err)
	assert.Equal(t, 204, deleteResp.HTTPStatus)

	// Check if the managed object was deleted
	moResp := client.ManagedObjects.Get(context.Background(), binary2.Data.ID(), managedobjects.GetOptions{})
	assert.Error(t, moResp.Err)
	assert.Equal(t, 404, moResp.HTTPStatus)

	// Check if the binary was deleted
	downloadedBinary3 := client.Binaries.Get(context.Background(), binary2.Data.ID())
	assert.Error(t, downloadedBinary3.Err, "Error should contain additional information about the request")
}

func TestInventoryService_CreateBinaryWithProgressBar(t *testing.T) {
	client := testcore.CreateTestClient(t)

	testfile1 := testcore.NewDummyFileWithSize(t, "testfile1.txt", 10_000_000)

	output := bytes.NewBufferString("")

	progress := mpb.New(
		mpb.WithOutput(output),
		mpb.WithWidth(120),
		mpb.WithAutoRefresh(),
	)
	var size int64
	fileInfo, err := os.Stat(testfile1)
	if err != nil {
		t.Error(err)
	}
	basename := filepath.Base(testfile1)
	size = fileInfo.Size()

	file, err := os.Open(testfile1)
	require.NoError(t, err)
	defer file.Close()

	bar := progress.New(
		int64(size),
		mpb.BarStyle().Lbound("[").Filler("━").Tip("━").Padding(" ").Rbound("]"),
		mpb.PrependDecorators(
			decor.Name("elapsed", decor.WC{W: len("elapsed") + 1, C: decor.DindentRight}),
			decor.Elapsed(decor.ET_STYLE_MMSS, decor.WC{W: 8, C: decor.DindentRight}),
			decor.Name(basename, decor.WC{W: len(basename) + 1, C: decor.DindentRight}),
		),
		mpb.AppendDecorators(
			decor.Percentage(decor.WC{W: 6, C: decor.DindentRight}),
			decor.CountersKibiByte("% .2f / % .2f"),
		),
	)

	// Upload the binary
	binary1 := client.Binaries.Create(context.Background(), core.UploadFileOptions{
		Reader:   bar.ProxyReader(file),
		FilePath: testfile1,
	})
	assert.NoError(t, binary1.Err)
	assert.Equal(t, 201, binary1.HTTPStatus)

	// Verify the upload succeeded
	assert.NotEmpty(t, binary1.Data.ID())
	assert.NotEmpty(t, binary1.Data.Name())
	assert.Greater(t, binary1.Data.Length(), int64(0))

	// Cleanup
	t.Cleanup(func() {
		client.Binaries.Delete(context.Background(), binary1.Data.ID())
	})

	// Check the progress bar
	progress.Wait()

	progressOutput := string(testcore.DecodeAnsi(testcore.MustReadAll(t, output)))

	assert.Contains(t, progressOutput, "100 %", "Should contain a progress percentage")
}
