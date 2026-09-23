package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type JSONB json.RawMessage

func EmptyJSONB() JSONB {
	return JSONB(json.RawMessage(`{}`))
}

func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte(`{}`), nil
	}
	return json.RawMessage(j).MarshalJSON()
}

func (j *JSONB) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		*j = EmptyJSONB()
		return nil
	}
	*j = JSONB(append((*j)[0:0], data...))
	return nil
}

func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	return string(j), nil
}

func (j *JSONB) Scan(value any) error {
	if value == nil {
		*j = EmptyJSONB()
		return nil
	}

	switch v := value.(type) {
	case []byte:
		*j = JSONB(append((*j)[0:0], v...))
		return nil
	case string:
		*j = JSONB(json.RawMessage(v))
		return nil
	default:
		return fmt.Errorf("unsupported JSONB scan type %T", value)
	}
}
