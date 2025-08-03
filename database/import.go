package database

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gorm.io/gorm"
)

func ImportSQLFile(db *gorm.DB, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var stmt strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "--") || strings.TrimSpace(line) == "" {
			continue // skip comments and empty lines
		}

		stmt.WriteString(line + "\n")
		if strings.HasSuffix(strings.TrimSpace(line), ";") {
			if err := db.Exec(stmt.String()).Error; err != nil {
				fmt.Printf("Error executing:\n%s\nError: %v\n", stmt.String(), err)
			}
			stmt.Reset()
		}
	}

	return scanner.Err()
}
