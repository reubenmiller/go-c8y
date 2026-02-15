package core

import (
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

type Result[A any] struct {
	Request  *resty.Request
	Response *resty.Response

	data A
}

func NewResult[A any](r *resty.Request) *Result[A] {
	return &Result[A]{
		Request: r,
	}
}

func (o *Result[A]) Data() (*Result[A], error) {
	result := new(A)
	o.Request.SetResult(result)
	resp, err := o.Request.Send()
	return &Result[A]{
		Response: resp,
	}, err
}

type Response struct {
	Request  *TryRequest
	Response *resty.Response
}

func (r *Response) List() []gjson.Result {
	items := r.JSON(r.Request.Property)
	if items.Exists() && items.IsArray() {
		return items.Array()
	}
	return []gjson.Result{}
}

func (r *Response) String() string {
	return r.Response.String()
}

func (r *Response) ForEach(p func(key gjson.Result, value gjson.Result) bool) {
	items := r.JSON(r.Request.Property)
	if items.Exists() && items.IsArray() {
		items.ForEach(p)
	}
}

func (r *Response) JSON(prop ...string) gjson.Result {
	// TODO: How should non json data be handled
	if !gjson.Valid(r.Response.String()) {
		return gjson.Parse("")
	}
	result := gjson.Parse(r.Response.String())
	if len(prop) > 0 {
		return result.Get(prop[0])
	}
	return result
}

func (r *Response) Unmarshal(data any) error {
	if r.Response == nil {
		return fmt.Errorf("response is nil")
	}
	dec := json.NewDecoder(r.Response.Body)
	dec.UseNumber()
	return dec.Decode(&data)
}
