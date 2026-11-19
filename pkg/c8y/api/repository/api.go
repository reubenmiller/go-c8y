package repository

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/configuration"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/firmware"
	software "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/software"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
)

func NewService(s *httpcore.Service) *Service {
	return &Service{
		svc:           *s,
		Software:      software.NewService(s),
		Firmware:      firmware.NewService(s),
		Configuration: configuration.NewService(s),
	}
}

// Service api to interact with repository items
type Service struct {
	svc           httpcore.Service
	Software      *software.Service
	Firmware      *firmware.Service
	Configuration *configuration.Service
}
