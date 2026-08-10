package project

import (
	"bytes"
	"encoding/json"
)

// optionalString preserves the distinction PATCH needs between an omitted
// field and an explicitly supplied null value.
//
//	omitted     => Set=false, preserve current value
//	null        => Set=true, Value=nil, clear current value
//	"some/path" => Set=true, Value!=nil, replace current value
type optionalString struct {
	Set   bool
	Value *string
}

func (o *optionalString) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		o.Value = nil
		return nil
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}

func (o optionalString) Resolve(existing *string) *string {
	if !o.Set {
		return existing
	}
	return o.Value
}
