package remoteaccess

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/remoteaccess/remoteaccess_configurations"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
)

func NewService(s *httpcore.Service) *Service {
	return &Service{
		svc:            *s,
		Configurations: remoteaccess_configurations.NewService(s),
	}
}

// Service provides api to managed Cloud Remote Access in Cumulocity
type Service struct {
	svc            httpcore.Service
	Configurations *remoteaccess_configurations.Service
}
