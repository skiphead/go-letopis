package repository

import (
	"database/sql"
	"fmt"
	"reflect"
	"time"
)

// NullHandler provides unified NULL handling for reading and writing
type NullHandler struct{}

// ToNullString converts a string to sql.NullString with configurable strategy
func (nh *NullHandler) ToNullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: value, Valid: true}
}

// FromNullString converts sql.NullString to string with proper zero value handling
func (nh *NullHandler) FromNullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// ToNullInt64 converts int64 to sql.NullInt64
func (nh *NullHandler) ToNullInt64(value int64, zeroIsNull bool) sql.NullInt64 {
	if zeroIsNull && value == 0 {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: value, Valid: true}
}

// FromNullInt64 converts sql.NullInt64 to int64
func (nh *NullHandler) FromNullInt64(ni sql.NullInt64) int64 {
	if ni.Valid {
		return ni.Int64
	}
	return 0
}

// ToNullTime converts time.Time to sql.NullTime
func (nh *NullHandler) ToNullTime(value time.Time, zeroIsNull bool) sql.NullTime {
	if zeroIsNull && value.IsZero() {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: value, Valid: true}
}

// FromNullTime converts sql.NullTime to time.Time
func (nh *NullHandler) FromNullTime(nt sql.NullTime) time.Time {
	if nt.Valid {
		return nt.Time
	}
	return time.Time{}
}

// ToNullBool converts bool to sql.NullBool
func (nh *NullHandler) ToNullBool(value bool) sql.NullBool {
	return sql.NullBool{Bool: value, Valid: true}
}

// FromNullBool converts sql.NullBool to bool
func (nh *NullHandler) FromNullBool(nb sql.NullBool) bool {
	return nb.Bool
}

// ToNullFloat64 converts float64 to sql.NullFloat64
func (nh *NullHandler) ToNullFloat64(value float64, zeroIsNull bool) sql.NullFloat64 {
	if zeroIsNull && value == 0 {
		return sql.NullFloat64{Valid: false}
	}
	return sql.NullFloat64{Float64: value, Valid: true}
}

// FromNullFloat64 converts sql.NullFloat64 to float64
func (nh *NullHandler) FromNullFloat64(nf sql.NullFloat64) float64 {
	if nf.Valid {
		return nf.Float64
	}
	return 0
}

// shouldTreatAsNull determines if a value should be treated as NULL
func (r *GenericRepository[T]) shouldTreatAsNull(field reflect.Value, nullable bool) bool {
	if !nullable {
		return false
	}

	switch field.Kind() {
	case reflect.String:
		if field.String() == "" {
			return true
		}
		return false
	case reflect.Ptr:
		return field.IsNil()
	default:
		return field.IsZero()
	}
}

// extractSQLNullValue extracts the actual value from sql.Null* types
func (r *GenericRepository[T]) extractSQLNullValue(field reflect.Value) (interface{}, bool) {
	switch field.Type() {
	case reflect.TypeOf(sql.NullString{}):
		ns := field.Interface().(sql.NullString)
		if !ns.Valid {
			return nil, true
		}
		return ns.String, true
	case reflect.TypeOf(sql.NullInt64{}):
		ni := field.Interface().(sql.NullInt64)
		if !ni.Valid {
			return nil, true
		}
		return ni.Int64, true
	case reflect.TypeOf(sql.NullTime{}):
		nt := field.Interface().(sql.NullTime)
		if !nt.Valid {
			return nil, true
		}
		return nt.Time, true
	case reflect.TypeOf(sql.NullBool{}):
		nb := field.Interface().(sql.NullBool)
		if !nb.Valid {
			return nil, true
		}
		return nb.Bool, true
	case reflect.TypeOf(sql.NullFloat64{}):
		nf := field.Interface().(sql.NullFloat64)
		if !nf.Valid {
			return nil, true
		}
		return nf.Float64, true
	default:
		return nil, false
	}
}

// entityToFieldMap converts entity to map of column -> value with NULL handling
func (r *GenericRepository[T]) entityToFieldMap(entity *T) (map[string]interface{}, error) {
	val := reflect.ValueOf(entity).Elem()
	fields := make(map[string]interface{})

	for column, fieldName := range r.meta.ColumnMap {
		fieldIdx := r.meta.FieldIndex[column]
		field := val.Field(fieldIdx)

		if !field.IsValid() {
			return nil, fmt.Errorf("invalid field %s for column %s", fieldName, column)
		}

		// Check if this is a sql.Null* type
		if extracted, ok := r.extractSQLNullValue(field); ok {
			fields[column] = extracted
			continue
		}

		// Check if should treat as NULL
		if r.shouldTreatAsNull(field, r.meta.NullableMap[column]) {
			fields[column] = nil
			continue
		}

		fields[column] = field.Interface()
	}

	return fields, nil
}

// ScanNullableString is a convenience method that uses NullHandler
func (r *GenericRepository[T]) ScanNullableString(value sql.NullString) string {
	nh := &NullHandler{}
	return nh.FromNullString(value)
}
