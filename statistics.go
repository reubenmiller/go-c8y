package c8y

import "sync"

// Statistics statistics about the returned rest response
type Statistics struct {
	CurrentPage *int `json:"currentPage"`
	PageSize    *int `json:"pageSize"`
	TotalPages  *int `json:"totalPages"`
}

// BaseResponse common response from all c8y api calls
type BaseResponse struct {
	Next       *string     `json:"next"`
	Self       *string     `json:"self"`
	Statistics *Statistics `json:"statistics"`
}

// GetAll returns all of the objects as detailed in the pagination object
func (resp *BaseResponse) GetAll() (interface{}, error) {

	var wg sync.WaitGroup
	totalPages := 10

	for i := 2; i < totalPages; i++ {
		wg.Add(1)

		// Send parallel calls to collect all of the devices
		go func() {
			defer wg.Done()
		}()
	}

	wg.Wait()

	return resp.Next, nil
}
