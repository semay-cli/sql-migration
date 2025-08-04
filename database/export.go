package database

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

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

	file.WriteString(fmt.Sprintf("INSERT INTO %s (%s) VALUES\n", tableName, strings.Join(cols, ", ")))

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
			case time.Time:
				// Format MySQL-compatible datetime string
				valueStrings = append(valueStrings, fmt.Sprintf("'%s'", v.Format("2006-01-02 15:04:05")))
			default:
				valueStrings = append(valueStrings, fmt.Sprintf("'%v'", v))
			}
		}

		stmt := fmt.Sprintf("(%s),\n",
			strings.Join(valueStrings, ", "),
		)

		file.WriteString(stmt)
	}

	return nil
}

func FixSQLFileLastComma(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Look for the last ",\n" and replace it with ";\n"
	target := []byte(",\n")
	replacement := []byte(";\n")
	idx := bytes.LastIndex(content, target)

	if idx == -1 {
		return fmt.Errorf("no trailing comma found in: %s", filePath)
	}

	copy(content[idx:], replacement)

	err = os.WriteFile(filePath, content, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Fixed last comma in: %s\n", filePath)
	return nil
}

func FixOrTruncateSQLFile(filePath string) error {
	// First, open and count the number of non-empty lines
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		if line := scanner.Text(); len(line) > 0 {
			lineCount++
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning file: %w", err)
	}

	// If only one non-empty line, truncate the file
	if lineCount <= 1 {
		err := os.WriteFile(filePath, []byte{}, 0644)
		if err != nil {
			return fmt.Errorf("failed to truncate file: %w", err)
		}
		fmt.Printf("File truncated (only one line): %s\n", filePath)
		return nil
	}

	// Otherwise, read content and fix the trailing comma
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	target := []byte(",\n")
	replacement := []byte(";\n")
	idx := bytes.LastIndex(content, target)

	if idx == -1 {
		return fmt.Errorf("no trailing comma found in: %s", filePath)
	}

	copy(content[idx:], replacement)
	err = os.WriteFile(filePath, content, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Fixed last comma in: %s\n", filePath)
	return nil
}
