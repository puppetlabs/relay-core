package jsonutil

import (
	"encoding/json"
	"net/url"
)

type URL struct {
	*url.URL
}

func (u *URL) String() string {
	if u.URL == nil {
		return ""
	}

	return u.URL.String()
}

func (u *URL) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.URL.String())
}

func (u *URL) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	up, err := url.Parse(s)
	if err != nil {
		return err
	}

	u.URL = up
	return nil
}
