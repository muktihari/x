package sqlmap

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/muktihari/x/sqlmap/opt"
)

var (
	// ErrRecordNotFound is error record not found
	ErrRecordNotFound = errors.New("record not found")
)

// All fetchs all sqlRows to given v.
func All(sqlRows *sql.Rows, v interface{}, options ...opt.Option) error {
	results := []map[string]interface{}{}
	for sqlRows.Next() {
		m, err := Map(sqlRows, options...)
		if err != nil {
			return err
		}
		results = append(results, m)
	}

	if len(results) == 0 {
		return ErrRecordNotFound
	}

	b, err := json.Marshal(results)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, v)
}

// One fetchs last value of *sql.Rows to given v.
// Use Map if you want to have control over the iteration of *sql.Rows.
func One(sqlRows *sql.Rows, v interface{}, options ...opt.Option) error {
	var err error
	m := map[string]interface{}{}
	for sqlRows.Next() {
		m, err = Map(sqlRows, options...)
		if err != nil {
			return err
		}
	}

	if len(m) == 0 {
		return ErrRecordNotFound
	}

	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, v)
}

// Map maps current cursor of *sql.Rows pointed to a row into map[string]interface{}.
// You are in control (responsible) for the cursor iteration.
func Map(sqlRows *sql.Rows, options ...opt.Option) (map[string]interface{}, error) {
	columnNames, err := sqlRows.Columns()
	if err != nil {
		return nil, err
	}
	columnTypes, err := sqlRows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	var n = len(columnNames)
	var buff = make([]interface{}, n)
	var buffPtr = make([]interface{}, n)
	for i := range columnNames {
		buffPtr[i] = &buff[i]
	}

	if err := sqlRows.Scan(buffPtr...); err != nil {
		return nil, err
	}

	m := map[string]interface{}{}
	for i, columnName := range columnNames {
		m[columnName] = buff[i]
		for _, option := range options {
			data, err := option(buff[i], columnName, columnTypes[i])
			if err != nil {
				return nil, err
			}
			m[columnName] = data
		}
	}

	return m, nil
}
