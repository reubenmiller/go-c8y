package firmware

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/firmware/firmwareitems"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/firmware/firmwarepatches"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/firmware/firmwareversions"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
)

const FragmentFirmware = "c8y_Firmware"
const FragmentFirmwareBinary = "c8y_FirmwareBinary"

func NewService(s *httpcore.Service) *Service {
	return &Service{
		Service:  *firmwareitems.NewService(s),
		Versions: firmwareversions.NewService(s),
		Patches:  firmwarepatches.NewService(s),
	}
}

// Service api to interact with firmware items
type Service struct {
	firmwareitems.Service

	Versions *firmwareversions.Service
	Patches  *firmwarepatches.Service
}
