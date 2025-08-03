package database

import (
	"fmt"
	"os"
	"strings"

	"gorm.io/gorm"
)

func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func ExportSchemaSQL(db *gorm.DB, tableName, filename string) error {
	var createStmt string

	switch db.Dialector.Name() {
	case "mysql":
		row := db.Raw(fmt.Sprintf("SHOW CREATE TABLE %s", tableName)).Row()
		var table, stmt string
		if err := row.Scan(&table, &stmt); err != nil {
			return err
		}
		createStmt = stmt
	case "postgres":
		// Build a partial CREATE TABLE statement from information_schema
		rows, err := db.Raw(`
            SELECT column_name, data_type, is_nullable, column_default
            FROM information_schema.columns
            WHERE table_name = ?
            ORDER BY ordinal_position
        `, tableName).Rows()
		if err != nil {
			return err
		}
		defer rows.Close()

		var columns []string
		for rows.Next() {
			var name, dataType, isNullable string
			var colDefault *string
			if err := rows.Scan(&name, &dataType, &isNullable, &colDefault); err != nil {
				return err
			}
			colDef := fmt.Sprintf("%s %s", name, dataType)
			if colDefault != nil {
				colDef += fmt.Sprintf(" DEFAULT %s", *colDefault)
			}
			if isNullable == "NO" {
				colDef += " NOT NULL"
			}
			columns = append(columns, colDef)
		}
		if len(columns) == 0 {
			return fmt.Errorf("no columns found for table %s", tableName)
		}
		createStmt = fmt.Sprintf("CREATE TABLE %s (\n    %s\n);", tableName, strings.Join(columns, ",\n    "))
	case "sqlite":
		query := fmt.Sprintf("SELECT sql FROM sqlite_master WHERE type='table' AND name='%s';", tableName)
		row := db.Raw(query).Row()
		if err := row.Scan(&createStmt); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported dialect: %s", db.Dialector.Name())
	}

	return os.WriteFile(filename, []byte(createStmt+"\n"), 0644)
}

func ExportDataSQL(db *gorm.DB, driver, tableName, filename string) error {
	rows, err := db.Table(tableName).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	colCount := len(cols)

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for rows.Next() {
		values := make([]any, colCount)
		ptrs := make([]any, colCount)
		for i := range values {
			ptrs[i] = &values[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			return err
		}

		var valueStrings []string
		for _, val := range values {
			switch v := val.(type) {
			case nil:
				valueStrings = append(valueStrings, "NULL")
			case []byte:
				s := string(v)
				valueStrings = append(valueStrings, "'"+escapeSQLString(s)+"'")
			default:
				valueStrings = append(valueStrings, fmt.Sprintf("'%v'", v))
			}
		}

		stmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);\n",
			tableName,
			strings.Join(cols, ", "),
			strings.Join(valueStrings, ", "),
		)

		file.WriteString(stmt)
	}

	return nil
}
