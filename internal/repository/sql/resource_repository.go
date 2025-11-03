package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	pqUniqueViolationErrCode = "23505" // PostgreSQL unique violation error code. See https://www.postgresql.org/docs/14/errcodes-appendix.html
)

type ResourceRepository struct {
	db *sql.DB
	tx *sql.Tx
}

func NewRepository(db *sql.DB) *ResourceRepository {
	return &ResourceRepository{db: db}
}

func newRepositoryWithTx(tx *sql.Tx) *ResourceRepository {
	return &ResourceRepository{tx: tx}
}

func (r ResourceRepository) getExecutor() executor {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

type executor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// getTableName extracts the table name from the resource using reflection
func getTableName(resource repository.Resource) string {
	t := reflect.TypeOf(resource)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	// Check if the type has a TableName method
	resourceValue := reflect.ValueOf(resource)
	if method := resourceValue.MethodByName("TableName"); method.IsValid() {
		results := method.Call(nil)
		if len(results) > 0 {
			return results[0].String()
		}
	}
	
	// Default: convert type name to lowercase plural
	name := t.Name()
	return strings.ToLower(name) + "s"
}

// getFieldsAndValues extracts struct fields and values using reflection
func getFieldsAndValues(resource repository.Resource) ([]string, []interface{}) {
	var fields []string
	var values []interface{}
	
	v := reflect.ValueOf(resource)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		
		// Get column name from gorm tag or use field name
		columnName := field.Tag.Get("gorm")
		if columnName != "" {
			// Extract column name from gorm tag
			for _, part := range strings.Split(columnName, ";") {
				if strings.HasPrefix(part, "column:") {
					columnName = strings.TrimPrefix(part, "column:")
					break
				}
			}
		} else {
			columnName = strings.ToLower(field.Name)
		}
		
		// Skip fields with primaryKey in gorm tag for updates
		if strings.Contains(field.Tag.Get("gorm"), "primaryKey") {
			// Still include for inserts
			fields = append(fields, columnName)
			values = append(values, value.Interface())
			continue
		}
		
		fields = append(fields, columnName)
		values = append(values, value.Interface())
	}
	
	return fields, values
}

// getPrimaryKey extracts the primary key field and value
func getPrimaryKey(resource repository.Resource) (string, interface{}) {
	v := reflect.ValueOf(resource)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if strings.Contains(field.Tag.Get("gorm"), "primaryKey") {
			columnName := field.Tag.Get("gorm")
			for _, part := range strings.Split(columnName, ";") {
				if strings.HasPrefix(part, "column:") {
					columnName = strings.TrimPrefix(part, "column:")
					return columnName, v.Field(i).Interface()
				}
			}
			return strings.ToLower(field.Name), v.Field(i).Interface()
		}
	}
	
	// Default to "id" if no primary key tag found
	if idField := v.FieldByName("ID"); idField.IsValid() {
		return "id", idField.Interface()
	}
	
	return "id", nil
}

func (r ResourceRepository) Create(ctx context.Context, resource repository.Resource) error {
	resource.InitMeta()
	
	tableName := getTableName(resource)
	fields, values := getFieldsAndValues(resource)
	
	// Build INSERT query
	placeholders := make([]string, len(fields))
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
	)
	
	_, err := r.getExecutor().ExecContext(ctx, query, values...)
	if err != nil {
		slog.Error("error creating resource", slog.Any("err", err))
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pqUniqueViolationErrCode {
			return &repository.UniqueConstraintError{Detail: pgError.Detail}
		}
		return fmt.Errorf("failed to create resource: %w", err)
	}
	return nil
}

func (r ResourceRepository) List(ctx context.Context, result any, query repository.Query) error {
	// Get table name from result slice element type
	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Ptr || resultValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("result must be a pointer to a slice")
	}
	
	sliceType := resultValue.Elem().Type()
	elemType := sliceType.Elem()
	
	// Create a temporary instance to get table name
	var tableName string
	if elemType.Kind() == reflect.Ptr {
		tempInstance := reflect.New(elemType.Elem()).Interface()
		if tn, ok := tempInstance.(interface{ TableName() string }); ok {
			tableName = tn.TableName()
		}
	} else {
		tempInstance := reflect.New(elemType).Interface()
		if tn, ok := tempInstance.(interface{ TableName() string }); ok {
			tableName = tn.TableName()
		}
	}
	
	if tableName == "" {
		return fmt.Errorf("could not determine table name")
	}
	
	// Build WHERE clause
	var whereClauses []string
	var args []interface{}
	argIndex := 1
	
	for field, value := range query.Values {
		switch value {
		case string(repository.NotEmpty):
			whereClauses = append(whereClauses, fmt.Sprintf("%s IS NOT NULL AND %s != ''", field, field))
		case string(repository.Empty):
			whereClauses = append(whereClauses, fmt.Sprintf("(%s IS NULL OR %s = '')", field, field))
		default:
			whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", field, argIndex))
			args = append(args, value)
			argIndex++
		}
	}
	
	if query.Paginator != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("%s != $%d", repository.IDField, argIndex))
		args = append(args, query.Paginator.LastID)
		argIndex++
		
		whereClauses = append(whereClauses, fmt.Sprintf("%s <= $%d", repository.CreatedAtField, argIndex))
		args = append(args, query.Paginator.LastCreatedAt)
		argIndex++
	}
	
	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}
	
	if query.Limit == 0 {
		query.Limit = repository.DefaultPaginationLimit
	}
	
	sqlQuery := fmt.Sprintf(
		"SELECT * FROM %s %s ORDER BY %s DESC, %s LIMIT $%d",
		tableName,
		whereClause,
		repository.CreatedAtField,
		repository.IDField,
		argIndex,
	)
	args = append(args, query.Limit)
	
	rows, err := r.getExecutor().QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}
	defer rows.Close()
	
	// Scan rows into result slice
	slice := resultValue.Elem()
	slice.Set(reflect.MakeSlice(sliceType, 0, 0))
	
	for rows.Next() {
		var elem reflect.Value
		if elemType.Kind() == reflect.Ptr {
			elem = reflect.New(elemType.Elem())
		} else {
			elem = reflect.New(elemType)
		}
		
		// Get scan destinations
		scanDest := make([]interface{}, 0)
		elemValue := elem
		if elemValue.Kind() == reflect.Ptr {
			elemValue = elemValue.Elem()
		}
		
		for i := 0; i < elemValue.NumField(); i++ {
			scanDest = append(scanDest, elemValue.Field(i).Addr().Interface())
		}
		
		if err := rows.Scan(scanDest...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}
		
		if elemType.Kind() == reflect.Ptr {
			slice.Set(reflect.Append(slice, elem))
		} else {
			slice.Set(reflect.Append(slice, elem.Elem()))
		}
	}
	
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}
	
	return nil
}

func (r ResourceRepository) Delete(ctx context.Context, resource repository.Resource) error {
	tableName := getTableName(resource)
	pkField, pkValue := getPrimaryKey(resource)
	
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", tableName, pkField)
	
	_, err := r.getExecutor().ExecContext(ctx, query, pkValue)
	if err != nil {
		slog.Error("error deleting resource", slog.Any("err", err))
		return fmt.Errorf("failed to delete resource: %w", err)
	}
	return nil
}

func (r ResourceRepository) Find(ctx context.Context, resource repository.Resource) (bool, error) {
	tableName := getTableName(resource)
	pkField, pkValue := getPrimaryKey(resource)
	
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1 LIMIT 1", tableName, pkField)
	
	v := reflect.ValueOf(resource)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	// Get scan destinations
	scanDest := make([]interface{}, 0)
	for i := 0; i < v.NumField(); i++ {
		scanDest = append(scanDest, v.Field(i).Addr().Interface())
	}
	
	err := r.getExecutor().QueryRowContext(ctx, query, pkValue).Scan(scanDest...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		slog.Error("error finding a resource", slog.Any("err", err))
		return false, fmt.Errorf("failed to find resource: %w", err)
	}

	return true, nil
}

func (r ResourceRepository) Patch(ctx context.Context, resource repository.Resource) (bool, error) {
	tableName := getTableName(resource)
	pkField, pkValue := getPrimaryKey(resource)
	fields, values := getFieldsAndValues(resource)
	
	// Build UPDATE query, excluding primary key from SET clause
	var setClauses []string
	var setValues []interface{}
	argIndex := 1
	
	for i, field := range fields {
		if field == pkField {
			continue
		}
		// Skip zero values for patch (similar to GORM Updates behavior)
		value := values[i]
		if isZeroValue(value) {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argIndex))
		setValues = append(setValues, value)
		argIndex++
	}
	
	if len(setClauses) == 0 {
		return false, nil
	}
	
	// Add updated_at field
	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIndex))
	setValues = append(setValues, time.Now())
	argIndex++
	
	// Add WHERE clause for primary key
	setValues = append(setValues, pkValue)
	
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = $%d",
		tableName,
		strings.Join(setClauses, ", "),
		pkField,
		argIndex,
	)
	
	result, err := r.getExecutor().ExecContext(ctx, query, setValues...)
	if err != nil {
		slog.Error("error patching resource", slog.Any("err", err))
		return false, fmt.Errorf("failed to patch resource: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

// isZeroValue checks if a value is a zero value
func isZeroValue(v interface{}) bool {
	if v == nil {
		return true
	}
	
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.String:
		return val.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return val.Float() == 0
	case reflect.Bool:
		return !val.Bool()
	case reflect.Struct:
		// Special handling for UUID and time.Time
		if _, ok := v.(uuid.UUID); ok {
			return v.(uuid.UUID) == uuid.Nil
		}
		if t, ok := v.(time.Time); ok {
			return t.IsZero()
		}
		return false
	default:
		return false
	}
}

// Transaction will give transaction locking on particular rows.
// txFunc is a type where we can define transaction logic.
// if txFunc return no error then transaction will be committed.
// else if txFunc return error then transaction will be rolled back.
// Note: don't use goroutines inside txFunc.
func (r ResourceRepository) Transaction(ctx context.Context, txFunc repository.TransactionFunc) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	
	errorChan := make(chan error, 1)
	go func() {
		errorChan <- txFunc(ctx, newRepositoryWithTx(tx))
	}()
	
	var txErr error
	select {
	case <-ctx.Done():
		txErr = ctx.Err()
	case txErr = <-errorChan:
	}
	
	if txErr != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction failed: %w, rollback failed: %v", txErr, rbErr)
		}
		return fmt.Errorf("transaction failed: %w", txErr)
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}
