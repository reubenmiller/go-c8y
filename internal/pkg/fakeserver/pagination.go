package fakeserver

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
)

// defaultPageSize is used when the caller doesn't specify pageSize.
const defaultPageSize = 5

// PaginationResult holds a page of items plus the collection envelope metadata.
type PaginationResult struct {
	Items         []json.RawMessage
	CurrentPage   int
	PageSize      int
	TotalPages    int
	TotalElements int
}

// Paginate extracts pageSize and currentPage from the request query parameters,
// then slices the items accordingly.
func Paginate(r *http.Request, items []json.RawMessage) PaginationResult {
	pageSize := queryInt(r, "pageSize", defaultPageSize)
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	currentPage := queryInt(r, "currentPage", 1)
	if currentPage < 1 {
		currentPage = 1
	}

	total := len(items)
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

	start := (currentPage - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	return PaginationResult{
		Items:         items[start:end],
		CurrentPage:   currentPage,
		PageSize:      pageSize,
		TotalPages:    totalPages,
		TotalElements: total,
	}
}

// BuildCollectionResponse builds the standard Cumulocity collection envelope.
//
//	{
//	  "<collectionKey>": [...],
//	  "self": "...",
//	  "statistics": { "currentPage": 1, "pageSize": 5 },
//	  "next": "..."
//	}
//
// totalPages/totalElements appear only when the request asked for them
// (withTotalPages/withTotalElements), and next is always present — both match
// the real platform, whose collections omit the (expensive) totals by default
// and carry a next link even on the last/empty page.
func BuildCollectionResponse(r *http.Request, baseURL string, collectionKey string, page PaginationResult) json.RawMessage {
	// Always emit an empty array (not null) for an empty page, matching real
	// Cumulocity. A nil slice would marshal to null and be mis-parsed as a
	// single element by collection models.
	if page.Items == nil {
		page.Items = []json.RawMessage{}
	}

	stats := map[string]int{
		"currentPage": page.CurrentPage,
		"pageSize":    page.PageSize,
	}
	if r.URL.Query().Get("withTotalPages") == "true" {
		stats["totalPages"] = page.TotalPages
	}
	if r.URL.Query().Get("withTotalElements") == "true" {
		stats["totalElements"] = page.TotalElements
	}

	selfURL := baseURL + r.URL.Path + "?pageSize=" + strconv.Itoa(page.PageSize) + "&currentPage=" + strconv.Itoa(page.CurrentPage)

	envelope := map[string]any{
		collectionKey: page.Items,
		"self":        selfURL,
		"statistics":  stats,
		"next":        baseURL + r.URL.Path + "?pageSize=" + strconv.Itoa(page.PageSize) + "&currentPage=" + strconv.Itoa(page.CurrentPage+1),
	}
	if page.CurrentPage > 1 {
		envelope["prev"] = baseURL + r.URL.Path + "?pageSize=" + strconv.Itoa(page.PageSize) + "&currentPage=" + strconv.Itoa(page.CurrentPage-1)
	}

	out, _ := json.Marshal(envelope)
	return out
}

// queryInt reads an integer query parameter with a default fallback.
func queryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
