package microservice

import (
	"bytes"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"

	_ "embed"
)

//go:embed testdata/cumulocity.json
var cumulocityManifestFile []byte

func Test_NewManifestFromJSON(t *testing.T) {
	manifest, err := NewManifest(nil, FromJSON(bytes.NewBuffer(cumulocityManifestFile)))
	testingutils.Ok(t, err)
	testingutils.Assert(t, manifest.APIVersion == "v2", "api version matches")
	// manifest.Name = "go-c8y-starter"
	// manifest.Version = "0.0.1-SNAPSHOT"
	// out, err := Build(BuildOptions{
	// 	Manifest:     *manifest,
	// 	DockerFile:   "go-c8y-starter/Dockerfile",
	// 	BuildContext: "go-c8y-starter",
	// })
	// testingutils.Ok(t, err)
	// testingutils.Assert(t, out != "", "filepath is not empty")
}
