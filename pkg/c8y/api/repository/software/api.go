package software

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/software/softwareitems"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/software/softwareversions"
)

var ApiManagedObjects = "/inventory/managedObjects"
var ApiManagedObject = "/inventory/managedObjects/{id}"

const ParamID = "id"

const ResultProperty = "managedObjects"

const FragmentSoftware = "c8y_Software"
const FragmentSoftwareBinary = "c8y_SoftwareBinary"

func NewService(s *core.Service) *Service {
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
