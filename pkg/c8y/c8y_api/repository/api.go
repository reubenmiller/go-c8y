package repository

import (
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	software "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/repository/software"
)

func NewService(s *core.Service) *Service {
	return &Service{
		Service:  *s,
		Software: software.NewService(s),
	}
}

// Service api to interact with repository items
type Service struct {
	core.Service

	Software *software.Service
}
