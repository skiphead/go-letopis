package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GenericRepository provides common database operations
type GenericRepository[T any] struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	meta   *EntityMeta
}

// NewGenericRepository creates a new generic repository instance
func NewGenericRepository[T any](db *pgxpool.Pool, logger *slog.Logger, tableName string) (*GenericRepository[T], error) {
	meta, err := getEntityMeta[T](tableName)
	if err != nil {
		return nil, err
	}

	return &GenericRepository[T]{
		pool:   db,
		logger: logger,
		meta:   meta,
	}, nil
}

// validateColumnName validates that column name is safe to use in SQL query
func (r *GenericRepository[T]) validateColumnName(column string) error {
	if !columnNameRegex.MatchString(column) {
		return fmt.Errorf("invalid column name: %s", column)
	}
	return nil
}

// validateColumns validates all column names
func (r *GenericRepository[T]) validateColumns(columns []string) error {
	for _, column := range columns {
		if err := r.validateColumnName(column); err != nil {
			return err
		}
	}
	return nil
}

// sanitizeTsQuery sanitizes tsquery input to prevent injection
func (r *GenericRepository[T]) sanitizeTsQuery(query string) string {
	sanitized := tsQuerySanitizer.ReplaceAllString(query, " ")
	sanitized = strings.TrimSpace(strings.Join(strings.Fields(sanitized), " "))
	return sanitized
}

// buildScanDestinations builds scan destinations for a specific entity instance
func (r *GenericRepository[T]) buildScanDestinations(entity *T, columns []string) ([]interface{}, error) {
	val := reflect.ValueOf(entity).Elem()

	dest := make([]interface{}, len(columns))
	for i, column := range columns {
		// Find field index for this column
		fieldIdx, ok := r.meta.FieldIndex[column]
		if !ok {
			return nil, fmt.Errorf("column %s not found in entity metadata", column)
		}

		field := val.Field(fieldIdx)
		if !field.IsValid() {
			return nil, fmt.Errorf("invalid field for column: %s", column)
		}

		if !field.CanAddr() {
			return nil, fmt.Errorf("cannot address field for column: %s", column)
		}

		dest[i] = field.Addr().Interface()
	}

	return dest, nil
}

// Create inserts a new entity record into the database
func (r *GenericRepository[T]) Create(ctx context.Context, entity *T) error {
	fields, err := r.entityToFieldMap(entity)
	if err != nil {
		return fmt.Errorf("failed to convert entity to fields: %w", err)
	}

	if len(fields) == 0 {
		return errors.New("no fields provided for insert")
	}

	columns := make([]string, 0, len(fields))
	values := make([]interface{}, 0, len(fields))

	// Extract columns and values
	for column, value := range fields {
		columns = append(columns, column)
		values = append(values, value)
	}

	// Validate column names
	if err := r.validateColumns(columns); err != nil {
		return fmt.Errorf("column validation failed: %w", err)
	}

	// Build placeholders
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	sqlQuery := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`,
		r.meta.TableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	r.logger.Debug("executing insert query",
		"table", r.meta.TableName,
		"columns_count", len(columns))

	_, err = r.pool.Exec(ctx, sqlQuery, values...)
	if err != nil {
		return fmt.Errorf("failed to create entity in table %s: %w", r.meta.TableName, err)
	}

	return nil
}

// CreateWithReturning inserts a new entity and returns the created entity
func (r *GenericRepository[T]) CreateWithReturning(ctx context.Context, entity *T, returningColumns []string) (*T, error) {
	fields, err := r.entityToFieldMap(entity)
	if err != nil {
		return nil, fmt.Errorf("failed to convert entity to fields: %w", err)
	}

	if len(fields) == 0 {
		return nil, errors.New("no fields provided for insert")
	}

	if len(returningColumns) == 0 {
		return nil, errors.New("no returning columns specified")
	}

	columns := make([]string, 0, len(fields))
	values := make([]interface{}, 0, len(fields))

	// Extract columns and values
	for column, value := range fields {
		columns = append(columns, column)
		values = append(values, value)
	}

	// Validate column names
	if err := r.validateColumns(columns); err != nil {
		return nil, fmt.Errorf("column validation failed: %w", err)
	}
	if err := r.validateColumns(returningColumns); err != nil {
		return nil, fmt.Errorf("returning column validation failed: %w", err)
	}

	// Build placeholders
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	sqlQuery := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) RETURNING %s`,
		r.meta.TableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(returningColumns, ", "))

	r.logger.Debug("executing insert with returning query",
		"table", r.meta.TableName,
		"columns_count", len(columns),
		"returning_columns_count", len(returningColumns))

	row := r.pool.QueryRow(ctx, sqlQuery, values...)

	var result T
	dest, err := r.buildScanDestinations(&result, returningColumns)
	if err != nil {
		return nil, fmt.Errorf("failed to build scan destinations: %w", err)
	}

	err = row.Scan(dest...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("no rows returned after insert")
		}
		return nil, fmt.Errorf("failed to scan returned entity: %w", err)
	}

	return &result, nil
}

// GetByID retrieves an entity by ID
func (r *GenericRepository[T]) GetByID(ctx context.Context, id int64) (*T, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid id: %d", id)
	}

	columns := strings.Join(r.meta.Columns, ", ")
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE id = $1`, columns, r.meta.TableName)

	row := r.pool.QueryRow(ctx, query, id)

	var t T
	dest, err := r.buildScanDestinations(&t, r.meta.Columns)
	if err != nil {
		return nil, fmt.Errorf("failed to build scan destinations: %w", err)
	}

	err = row.Scan(dest...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s with id %d not found", r.meta.TableName, id)
		}
		return nil, fmt.Errorf("failed to get t: %w", err)
	}

	return &t, nil
}

// List retrieves all entities with pagination
func (r *GenericRepository[T]) List(ctx context.Context, limit, offset int) ([]T, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	const maxLimit = 1000
	if limit > maxLimit {
		limit = maxLimit
	}

	columns := strings.Join(r.meta.Columns, ", ")
	query := fmt.Sprintf(`SELECT %s FROM %s ORDER BY id LIMIT $1 OFFSET $2`, columns, r.meta.TableName)

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}
	defer rows.Close()

	var entities []T
	for rows.Next() {
		var t T
		dest, err := r.buildScanDestinations(&t, r.meta.Columns)
		if err != nil {
			return nil, fmt.Errorf("failed to build scan destinations: %w", err)
		}

		err = rows.Scan(dest...)
		if err != nil {
			return nil, fmt.Errorf("failed to scan t: %w", err)
		}
		entities = append(entities, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return entities, nil
}

// Update updates an entity
func (r *GenericRepository[T]) Update(ctx context.Context, entity *T) error {
	fields, err := r.entityToFieldMap(entity)
	if err != nil {
		return fmt.Errorf("failed to convert entity to fields: %w", err)
	}

	// Get ID value
	val := reflect.ValueOf(entity).Elem()
	idField := val.FieldByName("ID")
	if !idField.IsValid() {
		return errors.New("entity has no ID field")
	}

	id := idField.Int()
	if id <= 0 {
		return fmt.Errorf("invalid id: %d", id)
	}

	// Store original ID and remove from fields to update
	originalID := id
	delete(fields, "id")

	if len(fields) == 0 {
		return errors.New("no fields to update")
	}

	// Build SET clause with validation to prevent overwriting ID
	setClauses := make([]string, 0, len(fields))
	values := make([]interface{}, 0, len(fields)+1)
	i := 1
	for column, value := range fields {
		// Extra safety: ensure we're not updating ID
		if column == "id" {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", column, i))
		values = append(values, value)
		i++
	}

	// Add ID as last parameter
	values = append(values, originalID)

	// Validate column names
	columns := make([]string, 0, len(fields))
	for column := range fields {
		if column != "id" {
			columns = append(columns, column)
		}
	}
	if err := r.validateColumns(columns); err != nil {
		return fmt.Errorf("column validation failed: %w", err)
	}

	query := fmt.Sprintf(`UPDATE %s SET %s WHERE id = $%d`,
		r.meta.TableName,
		strings.Join(setClauses, ", "),
		i)

	r.logger.Debug("executing update query",
		"table", r.meta.TableName,
		"id", originalID,
		"fields_count", len(setClauses))

	result, err := r.pool.Exec(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("failed to update entity: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("%s with id %d not found", r.meta.TableName, originalID)
	}

	return nil
}

// Delete deletes an entity by ID
func (r *GenericRepository[T]) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid id: %d", id)
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, r.meta.TableName)
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("%s with id %d not found", r.meta.TableName, id)
	}

	return nil
}
