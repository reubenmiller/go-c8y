package remoteaccess

import (
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/remoteaccess/remoteaccess_configurations"
)

func NewService(s *core.Service) *Service {
	return &Service{
		Service:        *s,
		Configurations: remoteaccess_configurations.NewService(s),
	}
}

// Service provides api to get/set/delete events in Cumulocity
type Service struct {
	core.Service
	Configurations *remoteaccess_configurations.Service
}
