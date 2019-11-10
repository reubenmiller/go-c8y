package cmd

import (
	"bytes"
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thedevsaddam/gojsonq"
)

type JSONFilters struct {
	Filters   []JSONFilter
	Selectors []string
	Pluck     string
}

func (f JSONFilters) Apply(jsonValue string, property string) []byte {
	return filterJSON(jsonValue, property, f, f.Selectors, f.Pluck)
}

func (f *JSONFilters) AddSelectors(props ...string) {
	f.Selectors = append(f.Selectors, props...)
}

func (f *JSONFilters) Add(property, operation, value string) {
	f.Filters = append(f.Filters, JSONFilter{
		Property:  property,
		Operation: operation,
		Value:     value,
	})
}

func newJSONFilters() *JSONFilters {
	return &JSONFilters{
		Filters:   make([]JSONFilter, 0),
		Selectors: make([]string, 0),
	}
}

type JSONFilter struct {
	Property  string
	Operation string
	Value     string
}

func filterJSON(jsonValue string, property string, filters JSONFilters, selectors []string, pluck string) []byte {
	var b bytes.Buffer

	jq := gojsonq.New().FromString(jsonValue)

	if property != "" {
		jq.From(property)
	}

	for _, query := range filters.Filters {
		jq.Where(query.Property, query.Operation, query.Value)
	}

	if len(selectors) > 0 {
		jq.Select(selectors...)
	}

	// format values
	if pluck != "" {
		if result, err := jq.PluckR(pluck); err == nil {
			if values, err := result.StringSlice(); err == nil {
				log.Printf("plucking values")
				output := strings.Join(values, "\n")
				return []byte(output)
				// for _, item := range values {
				// 	b.WriteString(item + "\n")
				// }
			}
			log.Printf("ERROR: %s", err)
			return b.Bytes()
		} else {
			log.Printf("ERROR: %s", err)
		}
	}

	jq.Writer(&b)
	return b.Bytes()
}

func addFilterFlag(cmd *cobra.Command, name string) {
	if name == "" {
		name = "filter"
	}
	cmd.Flags().StringSlice(name, nil, "filter")
	cmd.Flags().StringSlice("select", nil, "select")
	cmd.Flags().String("format", "", "format")
}

func getFilterFlag(cmd *cobra.Command, flagName string) *JSONFilters {
	filters := newJSONFilters()

	if cmd.Flags().Changed("select") {
		if props, err := cmd.Flags().GetStringSlice("select"); err == nil {
			filters.AddSelectors(props...)
		}
	}

	if cmd.Flags().Changed("format") {
		if prop, err := cmd.Flags().GetString("format"); err == nil {
			filters.Pluck = prop
		}
	}

	if cmd.Flags().Changed(flagName) {
		if rawFilters, err := cmd.Flags().GetStringSlice(flagName); err == nil {
			for _, item := range rawFilters {
				parts := strings.SplitN(item, "=", 2)
				if len(parts) != 2 {
					continue
				}
				filters.Add(parts[0], "contains", parts[1])
			}
		}
	}

	return filters
}
