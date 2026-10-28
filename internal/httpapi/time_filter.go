package httpapi

import (
	"fmt"
	"net/url"
	"time"
)

func timeBounds(query url.Values) (time.Time, time.Time, error) {
	var from, to time.Time
	for name, target := range map[string]*time.Time{"from": &from, "to": &to} {
		if values, exists := query[name]; exists {
			if len(values) != 1 {
				return from, to, fmt.Errorf("%s must be specified once", name)
			}
			value, err := time.Parse(time.RFC3339Nano, values[0])
			if err != nil || value.IsZero() {
				return from, to, fmt.Errorf("%s must be a non-zero RFC3339 timestamp", name)
			}
			*target = value
		}
	}
	if !from.IsZero() && !to.IsZero() && from.After(to) {
		return from, to, fmt.Errorf("from must not be later than to")
	}
	return from, to, nil
}
