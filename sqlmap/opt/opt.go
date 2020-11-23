package opt

import (
	"database/sql"
	"encoding/json"
)

// Option is an option func to manipulate value of a column. For example handling JSONB data type in PostgreSQL.
type Option func(data interface{}, columnName string, columnType *sql.ColumnType) (v interface{}, err error)

// HandleJSONB handle column type JSONB and parse the JSON-encoded data ([]byte) as map[string]interface{}.
func HandleJSONB(data interface{}, columnName string, columnType *sql.ColumnType) (interface{}, error) {
	if columnType.DatabaseTypeName() != "JSONB" {
		return data, nil
	}

	b, ok := data.([]byte)
	if !ok {
		return data, nil
	}

	jsonb := map[string]interface{}{}
	if err := json.Unmarshal(b, &jsonb); err != nil {
		return nil, err
	}
	return jsonb, nil
}
