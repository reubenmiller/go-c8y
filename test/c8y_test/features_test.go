package c8y_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
)

func TestFeaturesService_GetFeatures(t *testing.T) {
	client := createTestClient()

	ctx := context.Background()
	features, resp, err := client.Features.GetFeatures(ctx)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Assert(t, len(features) > 0, "Feature array should not be empty")

	// Get single feature
	feature, resp, err := client.Features.GetFeature(ctx, features[0].Key)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Assert(t, feature.Key == features[0].Key, "Feature key should match")
	testingutils.Assert(t, feature.Phase == features[0].Phase, "Feature phase should match")
	testingutils.Assert(t, feature.Active == features[0].Active, "Feature activation should match")
	testingutils.Assert(t, feature.TenantId == "", "Feature tenant id should not be set")
}
