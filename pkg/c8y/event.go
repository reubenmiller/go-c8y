package c8y

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

// EventService does something
type EventService service

// EventCollectionOptions todo
type EventCollectionOptions struct {
	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	// DateFrom Timestamp `url:"dateFrom,omitempty"`
	DateFrom string `url:"dateFrom,omitempty"`

	DateTo string `url:"dateTo,omitempty"`

	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	PaginationOptions
}

// Event todo
type Event struct {
	ID     string     `json:"id,omitempty"`
	Source *Source    `json:"source,omitempty"`
	Type   string     `json:"type,omitempty"`
	Text   string     `json:"text,omitempty"`
	Self   string     `json:"self,omitempty"`
	Time   *Timestamp `json:"time,omitempty"`

	// Allow access to custom fields
	Item gjson.Result `json:"-"`
}

// EventCollection todo
type EventCollection struct {
	*BaseResponse

	Events []Event `json:"events"`

	// Allow access to custom fields
	Items []gjson.Result `json:"-"`
}

// GetEvent returns a new event object
func (s *EventService) GetEvent(ctx context.Context, ID string) (*Event, *Response, error) {
	data := new(Event)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "event/events/" + ID,
		ResponseData: data,
	})
	return data, resp, err
}

// GetEvents returns a list of events based on given filters
func (s *EventService) GetEvents(ctx context.Context, opt *EventCollectionOptions) (*EventCollection, *Response, error) {
	data := new(EventCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "event/events",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// Create creates a new event object
func (s *EventService) Create(ctx context.Context, body interface{}) (*Event, *Response, error) {
	data := new(Event)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "event/events",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Update updates properties on an existing event
func (s *EventService) Update(ctx context.Context, ID string, body interface{}) (*Event, *Response, error) {
	data := new(Event)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "event/events/" + ID,
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Delete event by its ID
func (s *EventService) Delete(ctx context.Context, ID string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "event/events/" + ID,
	})
}

// DeleteEvents removes a collection of events based on the given filters
func (s *EventService) DeleteEvents(ctx context.Context, opt *EventCollectionOptions) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "event/events",
		Query:  opt,
	})
}

// DownloadBinary retrieves the binary attached to the given event
func (s *EventService) DownloadBinary(ctx context.Context, ID string) (filepath string, err error) {
	// TODO: consolodate this func and InventoryService.DownloadBinary
	// set event binary api
	client := s.client
	u := "event/events/" + ID + "/binaries"

	req, err := s.client.NewRequest("GET", u, "", nil)
	if err != nil {
		zap.S().Errorf("Could not create request. %s", err)
		return
	}

	req.Header.Set("Accept", "*/*")

	// Get the data
	resp, err := client.Do(ctx, req, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", resp.Status)
		return
	}

	// Create the file
	tempDir, err := ioutil.TempDir("", "go-c8y_")

	if err != nil {
		err = fmt.Errorf("Could not create temp folder. %s", err)
		return
	}

	filepath = path.Join(tempDir, "binary-"+ID)
	out, err := os.Create(filepath)
	if err != nil {
		filepath = ""
		return
	}
	defer out.Close()

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		filepath = ""
		return
	}

	return
}

// CreateBinary uploads a binary that should be associated with an event. Size of attachment cannot exceed 50MB
func (s *EventService) CreateBinary(ctx context.Context, filename string, ID string) (*EventBinary, *Response, error) {
	client := s.client

	values := map[string]io.Reader{
		"file": mustOpen(filename), // lets assume its this file
	}

	// set binary api
	u, _ := url.Parse(client.BaseURL.String())
	u.Path = path.Join(u.Path, "/event/events/"+ID+"/binaries")

	req, err := prepareMultipartRequest(u.String(), "POST", values)
	s.client.SetAuthorization(req)

	req.Header.Set("Accept", "application/json")

	if err != nil {
		err = errors.Wrap(err, "Could not create binary upload request object")
		zap.S().Error(err)
		return nil, nil, err
	}

	data := new(EventBinary)
	resp, err := client.Do(ctx, req, data)

	if err != nil {
		return nil, resp, err
	}

	return data, resp, nil
}

// EventBinary binary object associated with an event
type EventBinary struct {
	Self   string `json:"self"`
	Type   string `json:"type"`
	Source string `json:"source"`
	Length int64  `json:"length"`

	// Allow access to custom fields
	Item gjson.Result `json:"-"`
}

// UpdateBinary updates an existing binary associated with an event
func (s *EventService) UpdateBinary(ctx context.Context, ID, filename string) (*EventBinary, *Response, error) {
	binarydata, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer binarydata.Close()

	data := new(EventBinary)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "event/events/" + ID + "/binaries",
		Body:         binarydata,
		ResponseData: data,
	})
	return data, resp, err
}

// DeleteBinary removes binary file associated to an event
func (s *EventService) DeleteBinary(ctx context.Context, ID string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "event/events/" + ID + "/binaries",
	})
}
