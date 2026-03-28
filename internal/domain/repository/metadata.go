package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// EntityMeta holds metadata about an entity type
type EntityMeta struct {
	Type        reflect.Type
	TableName   string
	Columns     []string
	ColumnMap   map[string]string // db tag -> field name
	FieldIndex  map[string]int    // column name -> field index
	NullableMap map[string]bool   // column name -> is nullable
	mu          sync.RWMutex
}

// metadataCache caches entity metadata
var metadataCache = &sync.Map{}

// getEntityMeta returns cached metadata for entity type T
func getEntityMeta[T any](tableName string) (*EntityMeta, error) {
	// Validate table name
	if err := validateTableName(tableName); err != nil {
		return nil, err
	}

	var t T
	key := fmt.Sprintf("%T_%s", t, tableName)

	if cached, ok := metadataCache.Load(key); ok {
		return cached.(*EntityMeta), nil
	}

	meta := &EntityMeta{
		Type:        reflect.TypeOf(t),
		TableName:   tableName,
		ColumnMap:   make(map[string]string),
		FieldIndex:  make(map[string]int),
		NullableMap: make(map[string]bool),
	}

	// Parse struct tags
	for i := 0; i < meta.Type.NumField(); i++ {
		field := meta.Type.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag == "" {
			dbTag = strings.ToLower(field.Name)
		}

		// Check if field is nullable
		nullable := false
		if nullTag := field.Tag.Get("null"); nullTag == "true" {
			nullable = true
		}

		// Check field type for nullability using type comparison
		fieldType := field.Type
		if fieldType == reflect.TypeOf(sql.NullString{}) ||
			fieldType == reflect.TypeOf(sql.NullInt64{}) ||
			fieldType == reflect.TypeOf(sql.NullTime{}) ||
			fieldType == reflect.TypeOf(sql.NullBool{}) ||
			fieldType == reflect.TypeOf(sql.NullFloat64{}) ||
			fieldType.Kind() == reflect.Ptr {
			nullable = true
		}

		meta.Columns = append(meta.Columns, dbTag)
		meta.ColumnMap[dbTag] = field.Name
		meta.FieldIndex[dbTag] = i
		meta.NullableMap[dbTag] = nullable
	}

	metadataCache.Store(key, meta)
	return meta, nil
}

// validateTableName validates table name to prevent SQL injection
func validateTableName(tableName string) error {
	if tableName == "" {
		return errors.New("table name cannot be empty")
	}
	if !columnNameRegex.MatchString(tableName) {
		return fmt.Errorf("invalid table name: %s", tableName)
	}
	return nil
}
