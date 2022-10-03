package c8y_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/vbauerster/mpb/v6"
	"github.com/vbauerster/mpb/v6/decor"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/c8y/binary"
)

func getDevices(client *c8y.Client, name string, pageSize int) (*c8y.ManagedObjectCollection, *c8y.Response, error) {

	opt := &c8y.ManagedObjectOptions{
		Query: fmt.Sprintf("has(c8y_IsDevice) and (name eq '%s')", name),
		PaginationOptions: c8y.PaginationOptions{
			PageSize: pageSize,
		},
	}
	col, resp, err := client.Inventory.GetManagedObjects(context.Background(), opt)

	return col, resp, err
}

func TestInventoryService_GetDevices(t *testing.T) {
	client := createTestClient()

	pageSize := 1
	opt := &c8y.PaginationOptions{
		PageSize: pageSize,
	}
	data, _, _ := client.Inventory.GetDevices(context.Background(), opt)

	if len(data.Items) != pageSize {
		t.Errorf("Unexpected amount of devices found. want %d, got: %d", pageSize, len(data.Items))
	}

	deviceName := data.Items[0].Get("name")

	log.Printf("Device Name: %s\n", deviceName)
}

func TestInventoryService_AuthenticationToken(t *testing.T) {
	client := createTestClient()

	pageSize := 1
	opt := &c8y.PaginationOptions{
		PageSize: pageSize,
	}
	// Throw invalid credentials
	ctx := c8y.NewAuthorizationContext("test", "something", "value")
	_, resp, err := client.Inventory.GetDevices(ctx, opt)

	if resp.StatusCode() != 401 {
		t.Errorf("Expected unauthorized access response. want: 401, got: %d", resp.StatusCode())
	}

	if err == nil {
		t.Errorf("Function should have thrown an error. %s", err)
	}
}

func TestInventoryService_CreateUpdateDeleteBinary(t *testing.T) {
	client := createTestClient()

	testfile1 := NewDummyFile("testfile1", "test contents 1")
	testfile2 := NewDummyFile("testfile2", "test contents 2")

	defer func() {
		os.Remove(testfile1)
		os.Remove(testfile2)
	}()

	testfile1_r, err := os.Open(testfile1)
	testingutils.Ok(t, err)

	// Configure required properties
	binaryFile, err := binary.NewBinaryFile(
		binary.WithReader(testfile1_r),
		binary.WithName("filename"),
		binary.WithType("text/plain"),
	)
	testingutils.Ok(t, err)

	// Upload a new binary
	binary1, resp, err := client.Inventory.CreateBinary(context.Background(), binaryFile)
	testingutils.Ok(t, err)
	testingutils.Assert(t, binary1.ID != "", "Binary ID should not be an empty string")
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())

	// Download the binary, and check if it matches the file that was uploaded exactly
	downloadedBinary1, err := client.Inventory.DownloadBinary(context.Background(), binary1.ID)
	testingutils.Ok(t, err)
	defer os.Remove(downloadedBinary1)
	testingutils.FileEquals(t, testfile1, downloadedBinary1)

	// Update the existing binary with a new binary
	file2, err := os.Open(testfile2)
	testingutils.Ok(t, err)
	binary2, _, err := client.Inventory.UpdateBinary(context.Background(), binary1.ID, file2)
	testingutils.Ok(t, err)

	// testingutils.Assert(t, binary1.ID != binary2.ID, "Binary ID should change if the binary has been updated")
	testingutils.Assert(t, binary2.ID != "", "Binary id should not be an empty string")

	// Download the updated binary and check if it matches the new binary contents
	downloadedBinary2, err := client.Inventory.DownloadBinary(context.Background(), binary2.ID)
	testingutils.Ok(t, err)

	defer os.Remove(downloadedBinary2)
	testingutils.FileEquals(t, testfile2, downloadedBinary2)

	// Delete the binary
	resp, err = client.Inventory.DeleteBinary(context.Background(), binary2.ID)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode())

	// Check if the managed object was deleted
	_, resp, err = client.Inventory.GetManagedObject(context.Background(), binary2.ID, nil)
	testingutils.Equals(t, http.StatusNotFound, resp.StatusCode())
	testingutils.Assert(t, err != nil, "Error should contain additional information about the request")

	// Check if the binary was deleted
	downloadedBinary3, err := client.Inventory.DownloadBinary(context.Background(), binary2.ID)
	testingutils.Assert(t, err != nil, "Error should contain additional information about the request")
	testingutils.Equals(t, "", downloadedBinary3)
}

func TestInventoryService_CustomReader(t *testing.T) {
	client := createTestClient()

	testfile1 := NewDummyFile("testfile1", "test contents 1")

	defer func() {
		os.Remove(testfile1)
	}()

	file, err := os.Open(testfile1)
	testingutils.Ok(t, err)

	// Configure required properties
	binaryFile, err := binary.NewBinaryFile(
		binary.WithReader(file),
		binary.WithName("filename"),
		binary.WithType("text/plain"),
	)
	testingutils.Ok(t, err)

	binary1, resp, err := client.Inventory.CreateBinary(context.Background(), binaryFile)
	testingutils.Ok(t, err)
	defer func(ID string) {
		client.Inventory.DeleteBinary(context.Background(), ID)
	}(binary1.ID)
	testingutils.Assert(t, binary1.ID != "", "Binary ID should not be an empty string")
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Equals(t, "text/plain", binary1.Type)
	testingutils.Equals(t, "filename", binary1.Name)
}

func TestInventoryService_CreateManagedObjectWithBinary(t *testing.T) {
	client := createTestClient()

	testfile1 := NewDummyFile("testfile1.txt", "test contents 1")
	defer os.Remove(testfile1)

	file, err := os.Open(testfile1)
	testingutils.Ok(t, err)

	// Configure required properties
	binaryFile, err := binary.NewBinaryFile(
		binary.WithReader(file),
		binary.WithFileProperties(testfile1),
	)
	testingutils.Ok(t, err)

	binary1, resp, err := client.Inventory.CreateWithBinary(context.Background(), binaryFile, func(binaryURL string) interface{} {
		return map[string]interface{}{
			"name": "MyConfigurationFile",
			"url":  binaryURL,
		}
	})
	testingutils.Ok(t, err)

	defer func(ID string) {
		client.Inventory.DeleteBinary(context.Background(), ID)
	}(binary1.ID)

	testingutils.Assert(t, binary1.ID != "", "Binary ID should not be an empty string")
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())

	binaryURL := binary1.Item.Get("url").String()
	binaryID := binaryURL[strings.LastIndex(binaryURL, "/")+1:]

	defer func(ID string) {
		client.Inventory.DeleteBinary(context.Background(), ID)
	}(binaryID)

	binary, _, err := client.Inventory.GetManagedObject(context.Background(), binaryID, nil)
	testingutils.Ok(t, err)

	testingutils.Equals(t, "text/plain", binary.Type)
	testingutils.Equals(t, "testfile1.txt", binary.Name)

	testingutils.Equals(t, strings.ReplaceAll(binary.Self, "managedObjects", "binaries"), binary1.Item.Get("url").String())
}

func readOutput(t *testing.T, b io.Reader) string {
	out, err := ioutil.ReadAll(b)
	testingutils.Ok(t, err)
	return string(out)
}

func decodeAnsi(v string) string {
	ansi_escape := regexp.MustCompile("\x1B(?:[@-Z\\-_]|[[0-?]*[ -/]*[@-~])")
	return string(ansi_escape.ReplaceAll([]byte(v), []byte{}))
}

func TestInventoryService_CreateBinaryWithProgressBar(t *testing.T) {
	client := createTestClient()
	BarFiller := "[━━ ]"
	testfile1 := NewDummyFileWithSize("testfile1.txt", 10_000_000)
	defer os.Remove(testfile1)

	output := bytes.NewBufferString("")

	progress := mpb.New(
		mpb.WithOutput(output),
		mpb.WithWidth(120),
		mpb.WithRefreshRate(180*time.Millisecond),
	)
	var size int64
	fileInfo, err := os.Stat(testfile1)
	if err != nil {
		t.Error(err)
	}
	basename := filepath.Base(testfile1)
	size = fileInfo.Size()

	file, err := os.Open(testfile1)
	testingutils.Ok(t, err)

	bar := progress.Add(size,
		mpb.NewBarFiller(BarFiller),
		mpb.PrependDecorators(
			decor.Name("elapsed", decor.WC{W: len("elapsed") + 1, C: decor.DidentRight}),
			decor.Elapsed(decor.ET_STYLE_MMSS, decor.WC{W: 8, C: decor.DidentRight}),
			decor.Name(basename, decor.WC{W: len(basename) + 1, C: decor.DidentRight}),
		),
		mpb.AppendDecorators(
			decor.Percentage(decor.WC{W: 6, C: decor.DidentRight}),
			decor.CountersKibiByte("% .2f / % .2f"),
		),
	)
	binaryFile, err := binary.NewBinaryFile(
		binary.WithReader(file),
		binary.WithFileProperties(testfile1),
	)
	testingutils.Ok(t, err)

	_, resp, err := client.Inventory.CreateBinary(context.Background(), binaryFile, func(r *http.Request) (*http.Request, error) {
		r.Body = bar.ProxyReader(r.Body)
		return r, nil
	})
	testingutils.Ok(t, err)
	testingutils.Equals(t, resp.StatusCode(), 201)

	defer func(ID string) {
		client.Inventory.DeleteBinary(context.Background(), ID)
	}(resp.JSON("id").String())

	progress.Wait()

	progressOutput := decodeAnsi(readOutput(t, output))

	testingutils.Assert(t, strings.Contains(progressOutput, "100 %"), "Progress should contain progress")

}
func TestInventoryService_CreateChildAdditionWithBinary(t *testing.T) {
	client := createTestClient()

	testfile1 := NewDummyFile("testfile1.txt", "test contents 1")
	defer os.Remove(testfile1)

	file, err := os.Open(testfile1)
	testingutils.Ok(t, err)

	// Configure required properties
	binaryFile, err := binary.NewBinaryFile(
		binary.WithReader(file),
		binary.WithFileProperties(testfile1),
	)
	testingutils.Ok(t, err)

	// Create parent
	parentBody := map[string]interface{}{
		"name": "customParent",
	}
	parent, resp, err := client.Inventory.Create(context.Background(), parentBody)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Assert(t, parent != nil, "MO should have been created")

	// Create child addition and a binary
	childBody := map[string]interface{}{
		"name": "customChild",
	}
	child, resp, err := client.Inventory.CreateChildAdditionWithBinary(context.Background(), parent.ID, binaryFile, func(binaryURL string) interface{} {
		childBody["childUrl"] = binaryURL
		return childBody
	})
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())

	defer func(ID string) {
		deleteOptions := (&c8y.ManagedObjectDeleteOptions{}).WithForceCascade(true)
		client.Inventory.DeleteWithOptions(context.Background(), ID, deleteOptions)
	}(parent.ID)

	childURL := child.Item.Get("childUrl").String()
	testingutils.Assert(t, child.ID != "", "Addition ID should not be an empty string")
	testingutils.Assert(t, childURL != "", "Child url should not be an empty string")

	binaryID := childURL[strings.LastIndex(childURL, "/")+1:]
	binary, _, err := client.Inventory.GetManagedObject(context.Background(), binaryID, nil)
	testingutils.Ok(t, err)

	testingutils.Equals(t, "text/plain", binary.Type)
	testingutils.Equals(t, "testfile1.txt", binary.Name)
	testingutils.Equals(t, strings.ReplaceAll(binary.Self, "managedObjects", "binaries"), childURL)
}

func TestInventoryService_GetChildAdditions(t *testing.T) {
	client := createTestClient()

	device, err := TestEnvironment.NewRandomTestDevice()
	testingutils.Ok(t, err)
	opts := (&c8y.ManagedObjectDeleteOptions{}).WithCascade(true)
	defer client.Inventory.DeleteWithOptions(context.Background(), device.ID, opts)

	child01, _, err := client.Inventory.CreateChildAddition(
		context.Background(),
		device.ID,
		map[string]interface{}{
			"name": "ntp",
			"type": "c8y_Service",
		},
	)
	testingutils.Ok(t, err)

	child02, _, err := client.Inventory.CreateChildAddition(
		context.Background(),
		device.ID,
		map[string]interface{}{
			"name": "mosquitto",
			"type": "c8y_Service",
		},
	)
	testingutils.Ok(t, err)
	testingutils.Assert(t, child02.ID != "", "Id is not empty")

	items, _, err := client.Inventory.GetChildAdditions(
		context.Background(),
		device.ID,
		&c8y.ManagedObjectOptions{
			Query: "$filter=(type eq 'c8y_Service' and name eq 'ntp')",
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, 1, len(items.References))
	testingutils.Equals(t, items.References[0].ManagedObject.ID, child01.ID)
}
