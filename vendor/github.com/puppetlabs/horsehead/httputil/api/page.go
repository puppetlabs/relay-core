package api

import (
	"net/http"
	"net/url"
	"strconv"
)

type Page interface {
	Offset() string
	Limit() int
}

type defaultPage struct {
	offset string
	limit  int
}

func (dp *defaultPage) Offset() string {
	return dp.offset
}

func (dp *defaultPage) Limit() int {
	return dp.limit
}

func NewPage(offset string, limit int) Page {
	return &defaultPage{
		offset: offset,
		limit:  limit,
	}
}

func FirstPage(limit int) Page {
	return NewPage("", limit)
}

func NewPageFromRequest(r *http.Request, defaultLimit int) Page {
	offset := r.URL.Query().Get("offset")
	stringLimit := r.URL.Query().Get("limit")

	var limit int
	if parsedLimit, err := strconv.ParseInt(stringLimit, 10, 32); err == nil && parsedLimit > 0 {
		limit = int(parsedLimit)
	} else {
		limit = defaultLimit
	}

	return NewPage(offset, limit)
}

func SetPageQuery(out url.Values, r *http.Request) {
	out["offset"] = r.URL.Query()["offset"]
	out["limit"] = r.URL.Query()["limit"]
}
