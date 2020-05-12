package jsonutil

import "encoding/json"

type StrOrStrSlice []string

func (soss *StrOrStrSlice) UnmarshalJSON(data []byte) error {
	var delegate []string
	if err := json.Unmarshal(data, &delegate); err != nil {
		// Try to read it as a string instead.
		var fallback string
		if ferr := json.Unmarshal(data, &fallback); ferr != nil {
			// Nothing worked, so we return the original error.
			return err
		}

		delegate = append(delegate, fallback)
	}

	*soss = StrOrStrSlice(delegate)
	return nil
}
