package jsonmodels_test

import (
	"errors"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
)

func TestGetAs(t *testing.T) {
	doc := jsondoc.New([]byte(`{"type":"x","c8y_Position":{"lat":51.5,"lng":-0.1,"alt":10}}`))

	pos, err := jsonmodels.GetAs[model.Position](doc, "c8y_Position")
	if err != nil {
		t.Fatal(err)
	}
	if pos.Lat != 51.5 || pos.Lng != -0.1 || pos.Alt != 10 {
		t.Errorf("decoded position wrong: %+v", pos)
	}
}

func TestGetAsMissing(t *testing.T) {
	doc := jsondoc.New([]byte(`{"type":"x"}`))
	_, err := jsonmodels.GetAs[model.Position](doc, "c8y_Position")
	if !errors.Is(err, model.ErrFragmentNotFound) {
		t.Errorf("expected ErrFragmentNotFound, got %v", err)
	}
}

func TestGetFragmentUsesOwnKey(t *testing.T) {
	doc := jsondoc.New([]byte(`{"c8y_Hardware":{"serialNumber":"SN-1","model":"m"}}`))
	hw, err := jsonmodels.GetFragment[model.Hardware](doc)
	if err != nil {
		t.Fatal(err)
	}
	if hw.SerialNumber != "SN-1" || hw.Model != "m" {
		t.Errorf("decoded hardware wrong: %+v", hw)
	}
}

// TestWriteReadSymmetry confirms a typed fragment written via MergeFragments round-trips
// back through GetFragment.
func TestWriteReadSymmetry(t *testing.T) {
	body, err := model.MergeFragments([]byte(`{"type":"c8y_LocationUpdate"}`), []model.Fragment{
		model.Position{Lat: 1.25, Lng: 2.5, Alt: 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	pos, err := jsonmodels.GetFragment[model.Position](jsondoc.New(body))
	if err != nil {
		t.Fatal(err)
	}
	if pos.Lat != 1.25 || pos.Lng != 2.5 || pos.Alt != 3 {
		t.Errorf("symmetry broken: in {1.25,2.5,3}, out %+v", pos)
	}
}
