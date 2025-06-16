package cli

type Context struct {
	Debug    bool
	DeviceID string
	CertFile string
	KeyFile  string
	CAPath   string
	Host     string
	Port     int
}
