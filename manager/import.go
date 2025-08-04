package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/semay-cli/sql-migration/config"
	"github.com/semay-cli/sql-migration/database"
	"github.com/spf13/cobra"
)

// getAllSQLTables scans the directory for *_schema.sql or *_data.sql files and extracts table names.
func getAllSQLTables(dir string, suffix string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var tables []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, suffix) {
			tableName := strings.TrimSuffix(name, suffix)
			if tableName != "" {
				tables = append(tables, tableName)
			}
		}
	}

	return tables, nil
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import table schema and data from SQL files",
	Long:  "Import the schema and data of a specified table, or all tables if not specified, from .sql files in the exported folder.",
	Run: func(cmd *cobra.Command, args []string) {
		tableName, _ := cmd.Flags().GetString("table")
		inputDir, _ := cmd.Flags().GetString("input")
		env, _ := cmd.Flags().GetString("json")
		schemaOnly, _ := cmd.Flags().GetBool("schema-only")
		dataOnly, _ := cmd.Flags().GetBool("data-only")

		if inputDir == "" {
			inputDir = "exported"
		}

		// Load DSN config (dsn.json)
		configPath := "dsn.json"
		if env != "" {
			configPath = fmt.Sprintf("%v.json", env)
		}
		dsnCfg, err := config.LoadDSNConfig(configPath)
		if err != nil {
			fmt.Printf("Failed to load DSN config: %v\n", err)
			return
		}

		// Connect to import database
		db, err := database.ReturnSession("import", dsnCfg.Driver, dsnCfg.ImportDatabaseDSN)
		if err != nil {
			fmt.Printf("Failed to connect to database: %v\n", err)
			return
		}

		var tables []string
		if tableName == "" {
			// No table specified: import all tables found in the input directory
			if schemaOnly {
				tables, err = getAllSQLTables(inputDir, "_schema.sql")
				if err != nil {
					fmt.Printf("Failed to get table names from input directory: %v\n", err)
					return
				}
				if len(tables) == 0 {
					fmt.Println("No schema SQL files found in the input directory.")
					return
				}
				fmt.Printf("Importing all tables: %s\n", strings.Join(tables, ", "))
			}

			if dataOnly {
				tables, err = getAllSQLTables(inputDir, "_data.sql")
				if err != nil {
					fmt.Printf("Failed to get table names from input directory: %v\n", err)
					return
				}
				if len(tables) == 0 {
					fmt.Println("No schema SQL files found in the input directory.")
					return
				}
				fmt.Printf("Importing all tables: %s\n", strings.Join(tables, ", "))
			}

			// tables, err = getAllTableNames(dsnCfg.Driver, db)

		} else {
			tables = []string{tableName}
		}

		for _, tbl := range tables {
			if schemaOnly {
				schemaFile := filepath.Join(inputDir, fmt.Sprintf("%s_schema.sql", tbl))
				if _, err := os.Stat(schemaFile); err == nil {
					fmt.Printf("Importing schema from %s\n", schemaFile)
					if err := database.ImportSQLFile(db, schemaFile); err != nil {
						fmt.Printf("Failed to import schema for table %s: %v\n", tbl, err)
					} else {
						fmt.Printf("Schema imported for table %s\n", tbl)
					}
				}
			}
			if dataOnly {
				dataFile := filepath.Join(inputDir, fmt.Sprintf("%s_data.sql", tbl))
				if _, err := os.Stat(dataFile); err == nil {
					fmt.Printf("Importing data from %s\n", dataFile)
					if err := database.ImportSQLFile(db, dataFile); err != nil {
						fmt.Printf("Failed to import data for table %s: %v\n", tbl, err)
					} else {
						fmt.Printf("Data imported for table %s\n", tbl)
					}
				}
			}
		}
	},
}

func init() {
	importCmd.Flags().StringP("table", "T", "", "Table name to import (if not set, imports all tables found in input directory)")
	importCmd.Flags().StringP("input", "i", "exported", "Input directory for SQL files")
	importCmd.Flags().StringP("json", "j", "", "Specify json file name to load (e.g., dsn.json, defaults to dsn.json)")
	importCmd.Flags().Bool("schema-only", false, "Import only schema")
	importCmd.Flags().Bool("data-only", false, "Import only data")

	// Add the import command to your root command or application
	goFrame.AddCommand(importCmd)
}
