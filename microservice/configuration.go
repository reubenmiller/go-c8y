package microservice

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// NewConfiguration returns a new microservice configuration object
func NewConfiguration() *Configuration {
	return &Configuration{
		viper: viper.New(),
	}
}

// Configuration represents the micorservice's configuration
type Configuration struct {
	viper *viper.Viper
}

// InitConfiguration initiliases the configuration (i.e. reads from file / environment variables)
func (c *Configuration) InitConfiguration() {
	config := c.viper
	config.SetConfigName("application")
	config.AddConfigPath(".")

	err := config.ReadInConfig()

	if err != nil {
		zap.S().Warnf("Could not read application.properties file. %s", err)
	}

	// Set default settings
	zap.S().Debug("Setting default configuration properties")
	config.SetDefault("server.port", "80")
	config.SetDefault("application.name", "go-microservice")
	config.SetDefault("agent.identityType", "nx_Microservice")
	config.SetDefault("agent.operations.pollRate", "@every 1m")
	config.SetDefault("log.file", "/var/log/go-nifgate/app.log")

	// Enable all variables to be defined as (case-sensitive) environment variables in the form of
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	config.AutomaticEnv()

	// Add extra aliases for Cumulocity Microservice SDK Specific environment variables
	config.BindEnv("c8y.host", "C8Y_BASEURL")
	config.BindEnv("c8y.microservice.isolation", "C8Y_MICROSERVICE_ISOLATION")

	// Set proxy settings if defined. Otherwise the existing HTTP_PROXY and HTTPS_PROXY settings
	// will be honoured
	proxyHost := config.GetString("http.proxyHost")
	proxyPort := config.GetString("http.proxyPort")

	if proxyHost != "" && proxyPort != "" {
		zap.L().Info("Setting proxy")
		os.Setenv("HTTP_PROXY", fmt.Sprintf("http://%s:%s", proxyHost, proxyPort))
		os.Setenv("HTTPS_PROXY", fmt.Sprintf("http://%s:%s", proxyHost, proxyPort))
	}
}

// GetString retrieves a value from the configuration by it's key
func (c *Configuration) GetString(key string) string {
	return c.viper.GetString(key)
}

// AllKeys returns all of the keys in the configuration
func (c *Configuration) AllKeys() []string {
	return c.viper.AllKeys()
}

// SetDefault sets the default value for this key.
// SetDefault is case-insensitive for a key.
// Default only used when no value is provided by the user via flag, config or ENV.
func (c *Configuration) SetDefault(key, value string) {
	c.viper.SetDefault(key, value)
}

// GetConfigurationString returns the whole microservice configuration as text
func (c *Configuration) GetConfigurationString() string {
	var properties []string
	for _, key := range c.viper.AllKeys() {
		value := c.viper.GetString(key)
		properties = append(properties, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(properties, "\n")
}

// GetHost returns the configured Cumulocity Host
func (c *Configuration) GetHost() string {
	return c.viper.GetString("c8y.host")
}

// GetIdentityType returns the configured Cumulocity Host
func (c *Configuration) GetIdentityType() string {
	return c.viper.GetString("agent.identityType")
}

// GetApplicationName returns application's name
func (c *Configuration) GetApplicationName() string {
	return c.viper.GetString("application.name")
}

// GetMicroserviceHost returns the microservice host
// Either returns
func (c *Configuration) GetMicroserviceHost() (microserviceHost string) {
	// Get Microservice host address
	microserviceHost = c.viper.GetString("nx.microservice.host")

	if microserviceHost == "" {
		microserviceHost = fmt.Sprintf(
			"%s/service/%s",
			c.viper.GetString("c8y.host"),
			c.viper.GetString("application.name"),
		)
	} else {
		port := c.viper.GetString("server.port")
		if !strings.HasSuffix(microserviceHost, port) {
			microserviceHost += ":" + port
		}
	}
	return
}
