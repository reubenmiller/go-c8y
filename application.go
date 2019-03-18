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
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Self string `json:"self"`

	Item gjson.Result `json:"-"`
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
	u := partialURL

	var queryParams string
	var err error

	if opt != nil {
		queryParams, err = addOptions("", opt)
		if err != nil {
			return nil, nil, err
		}
	}

	req, err := s.client.NewRequest("GET", u, queryParams, nil)
	if err != nil {
		return nil, nil, err
	}

	data := new(ApplicationCollection)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	data.Items = resp.JSON.Get("applications").Array()
	return data, resp, nil
}

// GetApplicationCollectionByName returns a list of applications by name
func (s *ApplicationService) GetApplicationCollectionByName(ctx context.Context, name string, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	u := fmt.Sprintf("application/applicationsByName/%s", name)
	data, resp, err := s.getApplicationData(ctx, u, opt)
	return data, resp, err
}

// GetApplicationCollectionByOwner retuns a list of applications by owner
func (s *ApplicationService) GetApplicationCollectionByOwner(ctx context.Context, tenant string, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	u := fmt.Sprintf("appplication/applicationsByOwner/%s", tenant)
	return s.getApplicationData(ctx, u, opt)
}

// GetApplicationCollectionByTenant returns a list of applications by tenant name
func (s *ApplicationService) GetApplicationCollectionByTenant(ctx context.Context, tenant string, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	u := fmt.Sprintf("application/applicationsByTenant/%s", tenant)
	return s.getApplicationData(ctx, u, opt)
}

// GetApplicationCollectionByID returns an application by its ID
func (s *ApplicationService) GetApplicationCollectionByID(ctx context.Context, ID string) (*Application, *Response, error) {
	u := fmt.Sprintf("application/applications/%s", ID)

	var queryParams string
	var err error

	req, err := s.client.NewRequest("GET", u, queryParams, nil)
	if err != nil {
		return nil, nil, err
	}

	data := new(Application)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	data.Item = *resp.JSON

	return data, resp, nil
}

// GetApplications returns a list of applications with no filtering
func (s *ApplicationService) GetApplications(ctx context.Context, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	return s.getApplicationData(ctx, "/applications", opt)
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
