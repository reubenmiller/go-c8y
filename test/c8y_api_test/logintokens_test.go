package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/tenants/logintokens"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_LoginTokensCreate(t *testing.T) {
	client := testcore.CreateTestClient(t)

	tok := client.LoginTokens.Create(context.Background(), logintokens.CreateTokenOptions{
		Username:  authentication.GetEnvValue(authentication.EnvironmentUsername...),
		Password:  authentication.GetEnvValue(authentication.EnvironmentPassword...),
		GrantType: "PASSWORD",
	})
	assert.NoError(t, tok.Err)
	assert.NotEmpty(t, tok.Data.AccessToken())

	data := model.OAIToken{
		AccessToken: tok.Data.AccessToken(),
	}
	xsrfToken := data.GetXSRFToken()
	assert.NotEmpty(t, xsrfToken)
}

func Test_LoginTokensCreate_PermissionDenied(t *testing.T) {
	client := testcore.CreateTestClient(t)

	tok := client.LoginTokens.Create(context.Background(), logintokens.CreateTokenOptions{
		Username:  "test",
		Password:  "invalid",
		GrantType: "PASSWORD",
	})
	assert.Error(t, tok.Err)
	assert.True(t, c8y_api.ErrHasStatus(tok.Err, 401))
	assert.Equal(t, 0, tok.Data.Length())
}
