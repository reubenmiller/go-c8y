package remoteaccess

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/remoteaccess/remoteaccess_configurations"
)

func NewService(s *core.Service) *Service {
	return &Service{
		Service:        *s,
		Configurations: remoteaccess_configurations.NewService(s),
	}
}

// Service provides api to managed Cloud Remote Access in Cumulocity
type Service struct {
	core.Service
	Configurations *remoteaccess_configurations.Service
}
