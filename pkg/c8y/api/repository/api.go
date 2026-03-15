package repository

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/configuration"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/firmware"
	software "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/software"
)

func NewService(s *core.Service) *Service {
	return &Service{
		Service:       *s,
		Software:      software.NewService(s),
		Firmware:      firmware.NewService(s),
		Configuration: configuration.NewService(s),
	}
}

// Service api to interact with repository items
type Service struct {
	core.Service

	Software      *software.Service
	Firmware      *firmware.Service
	Configuration *configuration.Service
}
