package output_test

// Benchmarks that reproduce the go-c8y-cli pain case: a managed object
// collection of 2000 items at ~31MB total, which takes ~3s to process in
// go-c8y-cli's output pipeline. Run with:
//
//	go test -bench . -benchmem -run '^$' ./pkg/c8y/output/
//
// Throughput is reported in MB/s of response body processed; multiply the
// per-op time by 1 to compare directly against the CLI's ~3s.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output/encode"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output/filter"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output/shape"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output/template"
)

const benchItems = 2000

var (
	benchBody     []byte
	benchBodyOnce sync.Once
)

// fixture builds a managed object collection response of benchItems items,
// each ~15.5KB, mirroring the ~31MB / 2000 device response from the field.
func fixture() []byte {
	benchBodyOnce.Do(func() {
		var sb strings.Builder
		sb.Grow(33 << 20)
		sb.WriteString(`{"self": "https://example.com/inventory/managedObjects?pageSize=2000",`)
		sb.WriteString(`"next": "https://example.com/inventory/managedObjects?pageSize=2000&currentPage=2",`)
		sb.WriteString(`"managedObjects": [`)
		for i := range benchItems {
			if i > 0 {
				sb.WriteByte(',')
			}
			writeManagedObject(&sb, i)
		}
		sb.WriteString(`], "statistics": {"pageSize": 2000, "currentPage": 1}}`)
		benchBody = []byte(sb.String())
	})
	return benchBody
}

func writeManagedObject(sb *strings.Builder, i int) {
	osName := "linux"
	if i%3 == 0 {
		osName = "windows"
	}
	fmt.Fprintf(sb, `{"id": "%d", "name": "%s-device-%04d", "type": "c8y_%sDevice",`, 100000+i, osName, i, osName)
	fmt.Fprintf(sb, `"owner": "device_%s-device-%04d",`, osName, i)
	sb.WriteString(`"creationTime": "2025-04-01T10:00:00.000Z", "lastUpdated": "2026-06-11T09:30:00.000Z",`)
	sb.WriteString(`"c8y_IsDevice": {},`)
	fmt.Fprintf(sb, `"c8y_Hardware": {"model": "device-model-%d", "revision": "rev-%d", "serialNumber": "SN-%08d"},`, i%7, i%4, i)
	fmt.Fprintf(sb, `"c8y_Availability": {"status": "%s", "lastMessage": "2026-06-11T09:%02d:00.000Z"},`,
		map[bool]string{true: "AVAILABLE", false: "UNAVAILABLE"}[i%5 != 0], i%60)
	fmt.Fprintf(sb, `"c8y_ActiveAlarmsStatus": {"critical": %d, "major": %d, "minor": 0, "warning": 1},`, i%2, i%4)
	// Pad each object with a realistic configuration fragment to reach ~15.5KB.
	sb.WriteString(`"c8y_Configuration": {"items": [`)
	for j := range 100 {
		if j > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(sb,
			`{"key": "config/section-%02d/parameter-%02d", "value": "value-%d-%d", "unit": "s", "lastUpdated": "2026-06-10T08:00:00.000Z", "source": {"id": "%d", "kind": "device"}}`,
			j/10, j%10, i, j, 100000+i)
	}
	sb.WriteString(`]}}`)
}

func benchRender(b *testing.B, r func(io.Writer) output.Renderer, stages ...output.Stage) {
	body := fixture()
	b.SetBytes(int64(len(body)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := output.Render(context.Background(),
			output.FromBytes(body, "managedObjects"), r(io.Discard), stages...)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Baseline: split the collection into items and write them straight out.
func BenchmarkPassthroughNDJSON(b *testing.B) {
	benchRender(b, func(w io.Writer) output.Renderer { return encode.NewNDJSON(w) })
}

// Filter on a glob pattern (matches ~2/3 of items).
func BenchmarkFilterNDJSON(b *testing.B) {
	benchRender(b, func(w io.Writer) output.Renderer { return encode.NewNDJSON(w) },
		output.Filter(filter.Like("name", "linux*")))
}

// The typical CLI invocation: filter + select + table-ish output as CSV.
func BenchmarkFilterSelectCSV(b *testing.B) {
	benchRender(b, func(w io.Writer) output.Renderer {
		return encode.NewCSV(w, encode.CSVOptions{
			Header:  true,
			Columns: []string{"id", "name", "type", "c8y_Hardware.serialNumber", "c8y_Availability.status"},
		})
	}, output.Filter(filter.Like("name", "linux*")))
}

// Response shaping with both concrete paths and a wildcard pattern.
func BenchmarkSelectJSONArray(b *testing.B) {
	benchRender(b, func(w io.Writer) output.Renderer { return encode.NewJSONArray(w) },
		shape.Select("id", "name", "c8y_Hardware.*"))
}

// Streaming table with sampled column widths.
func BenchmarkTable(b *testing.B) {
	benchRender(b, func(w io.Writer) output.Renderer {
		return encode.NewTable(w, encode.TableOptions{
			Columns: []string{"id", "name", "type", "c8y_Hardware.serialNumber", "c8y_Availability.status"},
		})
	})
}

// Jsonnet response shaping: template compiled once (AST), evaluated per item.
func BenchmarkJsonnetTemplateNDJSON(b *testing.B) {
	stage, err := template.Jsonnet(
		`{id: output.id, name: output.name, serial: output.c8y_Hardware.serialNumber, alarms: output.c8y_ActiveAlarmsStatus}`)
	if err != nil {
		b.Fatal(err)
	}
	benchRender(b, func(w io.Writer) output.Renderer { return encode.NewNDJSON(w) }, stage)
}

// Streaming source: items decoded incrementally from a reader, as they would
// arrive from an HTTP response body.
func BenchmarkStreamingReaderNDJSON(b *testing.B) {
	body := fixture()
	b.SetBytes(int64(len(body)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := output.Render(context.Background(),
			output.FromReader(bytes.NewReader(body), "managedObjects"),
			encode.NewNDJSON(io.Discard))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Early exit: only the first 10 items are needed; the rest of the stream
// must not be processed (validates laziness on a large body).
func BenchmarkHead10FromReader(b *testing.B) {
	body := fixture()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := output.Render(context.Background(),
			output.FromReader(bytes.NewReader(body), "managedObjects"),
			encode.NewNDJSON(io.Discard),
			output.Head(10))
		if err != nil {
			b.Fatal(err)
		}
	}
}
