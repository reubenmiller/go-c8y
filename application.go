package c8y

import (
	"context"
	"fmt"

	"github.com/tidwall/gjson"
)

// ApplicationService provides the service provider for the Cumulocity Application API
type ApplicationService service

// ApplicationOptions options that can be provided when using application api calls
type ApplicationOptions struct {
	PaginationOptions
}

// Application todo
type Application struct {
	ID                string            `json:"id,omitempty"`
	Key               string            `json:"key,omitempty"`
	Name              string            `json:"name,omitempty"`
	Type              string            `json:"type,omitempty"`
	Availability      string            `json:"availability,omitempty"`
	Self              string            `json:"self,omitempty"`
	ContextPath       string            `json:"contextPath,omitempty"`
	ExternalURL       string            `json:"externalUrl,omitempty"`
	ResourcesURL      string            `json:"resourcesUrl,omitempty"`
	ResourcesUsername string            `json:"resourcesUsername,omitempty"`
	ResourcesPassword string            `json:"resourcesPassword,omitempty"`
	Owner             *ApplicationOwner `json:"owner,omitempty"`

	Item gjson.Result `json:"-"`
}

// ApplicationOwner application owner
type ApplicationOwner struct {
	Self   string                      `json:"self,omitempty"`
	Tenant *ApplicationTenantReference `json:"tenant,omitempty"`
}

// ApplicationTenantReference tenant reference information about the application
type ApplicationTenantReference struct {
	ID string `json:"id,omitempty"`
}

// ApplicationCollection contains information about a list of applications
type ApplicationCollection struct {
	*BaseResponse

	Applications []Application `json:"applications"`

	Items []gjson.Result `json:"-"`
}

// ApplicationSubscriptions contains the list of service users for each application subscription
type ApplicationSubscriptions struct {
	Users []ServiceUser `json:"users"`

	Item gjson.Result `json:"-"`
}

// ServiceUser has the service user credentials for a given application subscription
type ServiceUser struct {
	Username string `json:"name"`
	Password string `json:"password"`
	Tenant   string `json:"tenant"`
}

// getApplicationData todo
func (s *ApplicationService) getApplicationData(ctx context.Context, partialURL string, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	data := new(ApplicationCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         partialURL,
		Query:        opt,
		ResponseData: data,
	})
	data.Items = resp.JSON.Get("applications").Array()
	return data, resp, err
}

// GetApplicationsByName returns a list of applications by name
func (s *ApplicationService) GetApplicationsByName(ctx context.Context, name string, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	u := fmt.Sprintf("application/applicationsByName/%s", name)
	data, resp, err := s.getApplicationData(ctx, u, opt)
	return data, resp, err
}

// GetApplicationsByOwner retuns a list of applications by owner
func (s *ApplicationService) GetApplicationsByOwner(ctx context.Context, tenant string, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	u := fmt.Sprintf("application/applicationsByOwner/%s", tenant)
	return s.getApplicationData(ctx, u, opt)
}

// GetApplicationsByTenant returns a list of applications by tenant name
func (s *ApplicationService) GetApplicationsByTenant(ctx context.Context, tenant string, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	u := fmt.Sprintf("application/applicationsByTenant/%s", tenant)
	return s.getApplicationData(ctx, u, opt)
}

// GetApplication returns an application by its ID
func (s *ApplicationService) GetApplication(ctx context.Context, ID string) (*Application, *Response, error) {
	data := new(Application)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "application/applications/" + ID,
		ResponseData: data,
	})
	data.Item = *resp.JSON
	return data, resp, err
}

// GetApplications returns a list of applications with no filtering
func (s *ApplicationService) GetApplications(ctx context.Context, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	return s.getApplicationData(ctx, "/application/applications", opt)
}

// GetCurrentApplicationSubscriptions returns the list of application subscriptions per tenant along with the service user credentials
// This function can only be called using Application Bootstrap credentials, otherwise a 403 (forbidden) response will be returned
func (s *ApplicationService) GetCurrentApplicationSubscriptions(ctx context.Context) (*ApplicationSubscriptions, *Response, error) {
	data := new(ApplicationSubscriptions)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "application/currentApplication/subscriptions",
		ResponseData: data,
	})
	data.Item = *resp.JSON
	return data, resp, err
}
