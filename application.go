package c8y

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/tidwall/gjson"
)

// ApplicationService does something
type ApplicationService service

// ApplicationOptions todo
type ApplicationOptions struct {
	PaginationOptions
}

// Application todo
type Application struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Self string `json:"self"`

	Item gjson.Result
}

// ApplicationCollection todo
type ApplicationCollection struct {
	*BaseResponse

	Applications []Application `json:"applications"`

	Items []gjson.Result
}

// getApplicationData todo
func (s *ApplicationService) getApplicationData(ctx context.Context, partialURL string, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	if !strings.HasPrefix(partialURL, "/") {
		partialURL = "/" + partialURL
	}
	u := fmt.Sprintf("application/applications%s", partialURL)

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

	log.Printf("Total applicaitons: %d\n", *data.BaseResponse.Statistics.TotalPages)

	return data, resp, nil
}

// GetApplicationCollectionByName todo
func (s *ApplicationService) GetApplicationCollectionByName(ctx context.Context, name string, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	u := fmt.Sprintf("/applicationByName/%s", name)

	data, resp, err := s.getApplicationData(ctx, u, opt)

	if err != nil {
		return nil, nil, err
	}

	data.Items = resp.JSON.Get("applications").Array()

	return data, resp, nil
}

// GetApplicationCollectionByOwner todo
func (s *ApplicationService) GetApplicationCollectionByOwner(ctx context.Context, tenant string, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	u := fmt.Sprintf("/applicationsByOwner/%s", tenant)

	data, resp, err := s.getApplicationData(ctx, u, opt)

	if err != nil {
		return nil, nil, err
	}

	data.Items = resp.JSON.Get("applications").Array()

	return data, resp, nil
}

// GetApplicationCollectionByTenant todo
func (s *ApplicationService) GetApplicationCollectionByTenant(ctx context.Context, tenant string, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	u := fmt.Sprintf("/applicationsByTenant/%s", tenant)

	data, resp, err := s.getApplicationData(ctx, u, opt)

	if err != nil {
		return nil, nil, err
	}

	data.Items = resp.JSON.Get("applications").Array()

	return data, resp, nil
}

// GetApplicationCollectionByID todo
func (s *ApplicationService) GetApplicationCollectionByID(ctx context.Context, ID string) (*ApplicationCollection, *Response, error) {
	u := fmt.Sprintf("/%s", ID)

	data, resp, err := s.getApplicationData(ctx, u, nil)

	if err != nil {
		return nil, nil, err
	}

	data.Items = resp.JSON.Get("applications").Array()

	return data, resp, nil
}

// GetApplicationCollection todo
func (s *ApplicationService) GetApplicationCollection(ctx context.Context, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	data, resp, err := s.getApplicationData(ctx, "/applications", opt)

	if err != nil {
		return nil, nil, err
	}

	data.Items = resp.JSON.Get("applications").Array()

	return data, resp, nil
}
