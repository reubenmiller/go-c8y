package software

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/software/softwareitems"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/software/softwareversions"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
)

var apiManagedObjects = "/inventory/managedObjects"
var apiManagedObject = "/inventory/managedObjects/{id}"

const paramID = "id"

const resultProperty = "managedObjects"

const FragmentSoftware = "c8y_Software"
const FragmentSoftwareBinary = "c8y_SoftwareBinary"

func NewService(s *httpcore.Service) *Service {
	return &Service{
		Service:  *softwareitems.NewService(s),
		Versions: softwareversions.NewService(s),
	}
}

// Service api to interact with software items
type Service struct {
	softwareitems.Service

	Versions *softwareversions.Service
}
