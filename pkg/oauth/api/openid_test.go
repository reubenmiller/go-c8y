package api

import (
	"net/url"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
)

func Test_GetOpenIDFromMicrosoftURL(t *testing.T) {
	inputUrl, err := url.Parse(`https://login.microsoftonline.com/ec2515ec-0e38-48c0-bce2-f73789ba8326/oauth2/authorize?response_type=code&redirect_uri=https%3A%2F%2Fmain.dm-zz-d.ioee10-cloud.com%2Ftenant%2Foauth&client_id=7ebcfbcd-0db7-433c-86bb-90cd2d39b98d&sso_reload=true`)
	testingutils.Ok(t, err)
	configUrl := GetOpenIDConnectConfigurationURL(inputUrl)
	testingutils.Equals(t, "/ec2515ec-0e38-48c0-bce2-f73789ba8326/v2.0/.well-known/openid-configuration", configUrl)
}

func Test_GetOpenIDFromKeycloakURL(t *testing.T) {
	inputUrl, err := url.Parse(`https://mycustom.server.org/realms/example/oauth2/authorize`)
	testingutils.Ok(t, err)
	configUrl := GetOpenIDConnectConfigurationURL(inputUrl)
	testingutils.Equals(t, "/realms/example/.well-known/openid-configuration", configUrl)
}
