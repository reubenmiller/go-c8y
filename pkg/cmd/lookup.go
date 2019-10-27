package cmd

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type entityReference struct {
	ID   string           `json:"id,omitempty"`
	Name string           `json:"name,omitempty"`
	Data fetcherResultSet `json:"data,omitempty"`
}

type fetcherResultSet struct {
	ID    string      `json:"id,omitempty"`
	Name  string      `json:"name,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

func expandAlarms() error {
	return nil
}

type alarmFetcher struct {
	client *c8y.Client
}

func newAlarmFetcher(client *c8y.Client) *alarmFetcher {
	return &alarmFetcher{
		client: client,
	}
}

func (f *alarmFetcher) getByID(id string) (*fetcherResultSet, error) {
	_, resp, err := client.Alarm.GetAlarm(context.Background(), id)

	if err != nil {
		return nil, errors.Wrap(err, "Could not fetch by id")
	}

	return &fetcherResultSet{
		ID:    id,
		Value: resp.JSONData,
	}, nil
}

type deviceFetcher struct {
	client *c8y.Client
}

func newDeviceFetcher(client *c8y.Client) *deviceFetcher {
	return &deviceFetcher{
		client: client,
	}
}

func (f *deviceFetcher) getByID(id string) ([]fetcherResultSet, error) {
	mo, resp, err := client.Inventory.GetManagedObject(
		context.Background(),
		id,
		nil,
	)

	if err != nil {
		return nil, errors.Wrap(err, "Could not fetch by id")
	}

	results := make([]fetcherResultSet, 1)
	results[0] = fetcherResultSet{
		ID:    mo.ID,
		Name:  mo.Name,
		Value: *resp.JSON,
	}
	return results, nil
}

// getFormattedDeviceSlice returns the device id and name
// returns raw strings, lookuped values, and errors
func getFormattedDeviceSlice(cmd *cobra.Command, args []string, name string) ([]string, []string, error) {
	f := newDeviceFetcher(client)

	if !cmd.Flags().Changed(name) {
		// TODO: Read from os.PIPE
		pipedInput, err := getPipe()
		if err != nil {
			log.Printf("No pipeline input detected")
		} else {
			fmt.Printf("PIPED Input: %s\n", pipedInput)
			return nil, nil, nil
		}
	}

	values, err := cmd.Flags().GetStringSlice(name)
	if err != nil {
		log.Println("Flag is missing", err)
	}

	values = ParseValues(append(values, args...))

	formattedValues, err := lookupEntity(f, values, true)

	if err != nil {
		log.Printf("Failed to fetch entities. %s", err)
		return values, nil, err
	}

	results := []string{}

	invalidLookups := []string{}
	for _, item := range formattedValues {
		if item.ID != "" {
			if item.Name != "" {
				results = append(results, fmt.Sprintf("%s|%s", item.ID, item.Name))
			} else {
				results = append(results, item.ID)
			}
		} else {
			if item.Name != "" {
				invalidLookups = append(invalidLookups, item.Name)
			}
		}
	}

	var errors error

	if len(invalidLookups) > 0 {
		errors = fmt.Errorf("no results %v", invalidLookups)
	}

	return values, results, errors
}

type idValue struct {
	raw string
}

// newIDValue returns a new id formatter
// Example: newIDValue("12345|deviceName")
func newIDValue(raw string) *idValue {
	return &idValue{
		raw: raw,
	}
}

func (i *idValue) GetID() string {
	parts := strings.Split(i.raw, "|")
	return parts[0]
}

func (i *idValue) GetName() string {
	parts := strings.Split(i.raw, "|")

	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

func (f *deviceFetcher) getByName(name string) ([]fetcherResultSet, error) {
	mcol, _, err := client.Inventory.GetDevicesByName(
		context.Background(),
		name,
		c8y.NewPaginationOptions(5),
	)

	if err != nil {
		return nil, errors.Wrap(err, "Could not fetch by id")
	}

	results := make([]fetcherResultSet, len(mcol.ManagedObjects))

	for i, device := range mcol.ManagedObjects {
		results = append(results, fetcherResultSet{
			ID:    device.ID,
			Name:  device.Name,
			Value: mcol.Items[i],
		})
	}

	return results, nil
}

type entityFetcher interface {
	getByID(string) ([]fetcherResultSet, error)
	getByName(string) ([]fetcherResultSet, error)
}

func lookupEntity(fetch entityFetcher, values []string, getID bool) ([]entityReference, error) {
	ids, names := parseAndSanitizeIDs(values)

	entities := make([]entityReference, 0)

	// Lookup by id
	for _, id := range ids {
		if getID {
			if v, err := fetch.getByID(id); err == nil {
				for _, resultSet := range v {
					entities = append(entities, entityReference{
						ID:   id,
						Name: resultSet.Name,
						Data: resultSet,
					})
				}
			} else {
				// TODO: Handle error
				log.Printf("Failed to get entity by id. %s", err)
			}
		} else {
			entities = append(entities, entityReference{
				ID: id,
			})
		}

	}

	// Lookup via a name
	for _, name := range names {
		if v, err := fetch.getByName(name); err == nil {
			for _, resultSet := range v {
				entities = append(entities, entityReference{
					ID:   resultSet.ID,
					Name: resultSet.Name,
					Data: resultSet,
				})
			}
		} else {
			// TODO: Handle error
			log.Printf("Failed to get entity by id. %s", err)
		}
	}

	return entities, nil
}

func parseAndSanitizeIDs(values []string) (ids []string, names []string) {
	for _, value := range values {
		parts := strings.Split(strings.ReplaceAll(value, " ", ","), ",")

		for _, part := range parts {
			// Only add uint looking values
			if _, err := strconv.ParseUint(part, 10, 64); part != "" && err == nil {
				ids = append(ids, part)
			} else {
				names = append(names, part)
			}
		}
	}
	return
}
