package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/semay-cli/sql-migration/config"
	"github.com/semay-cli/sql-migration/database"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

// getAllTableNames retrieves all table names for the current database connection.
func getAllTableNames(driver string, db *gorm.DB) ([]string, error) {
	var tables []string
	switch driver {
	case "mysql":
		rows, err := db.Raw("SHOW TABLES").Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				return nil, err
			}
			tables = append(tables, table)
		}
	case "postgres":
		rows, err := db.Raw(`SELECT tablename FROM pg_tables WHERE schemaname = 'public'`).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				return nil, err
			}
			tables = append(tables, table)
		}
	case "sqlite":
		rows, err := db.Raw(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				return nil, err
			}
			tables = append(tables, table)
		}
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}
	return tables, nil
}

// exportCmd is the Cobra command for exporting a table's schema and data.
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export table schema and data to SQL files",
	Long:  "Export the schema and data of a specified table, or all tables if not specified, to .sql files in the exported folder.",
	Run: func(cmd *cobra.Command, args []string) {
		// Get flags
		tableName, _ := cmd.Flags().GetString("table")
		outputDir, _ := cmd.Flags().GetString("output")
		env, _ := cmd.Flags().GetString("json")
		schemaOnly, _ := cmd.Flags().GetBool("schema-only")
		dataOnly, _ := cmd.Flags().GetBool("data-only")

		// Set default output directory if not provided
		if outputDir == "" {
			outputDir = "exported"
		}

		// Ensure output directory exists
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("Failed to create output directory: %v\n", err)
			return
		}

		// Load DSN config (dsn.json)
		configPath := "dsn.json"
		if env != "" {
			configPath = fmt.Sprintf("%s.json", env)
		}
		dsnCfg, err := config.LoadDSNConfig(configPath)
		if err != nil {
			fmt.Printf("Failed to load DSN config: %v\n", err)
			return
		}

		// Connect to export database
		db, err := database.ReturnSession("export", dsnCfg.Driver, dsnCfg.ExportDatabaseDSN)
		if err != nil {
			fmt.Printf("Failed to connect to database: %v\n", err)
			return
		}

		var tables []string
		if tableName == "" {
			// No table specified: export all tables
			tables, err = getAllTableNames(dsnCfg.Driver, db)
			if err != nil {
				fmt.Printf("Failed to get table names: %v\n", err)
				return
			}
			if len(tables) == 0 {
				fmt.Println("No tables found in the database.")
				return
			}
			fmt.Printf("Exporting all tables: %s\n", strings.Join(tables, ", "))
		} else {
			// Export only the specified table
			tables = []string{tableName}
		}

		// Export schema and/or data for each table
		for _, tbl := range tables {
			if !dataOnly {
				schemaFile := filepath.Join(outputDir, fmt.Sprintf("%s_schema.sql", tbl))
				if err := database.ExportSchemaSQL(db, tbl, schemaFile); err != nil {
					fmt.Printf("Failed to export schema for table %s: %v\n", tbl, err)
				} else {
					fmt.Printf("Schema exported to %s\n", schemaFile)
				}
			}
			if !schemaOnly {
				dataFile := filepath.Join(outputDir, fmt.Sprintf("%s_data.sql", tbl))
				if err := database.ExportDataSQL(db, dsnCfg.Driver, tbl, dataFile); err != nil {
					fmt.Printf("Failed to export data for table %s: %v\n", tbl, err)
				} else {
					fmt.Printf("Data exported to %s\n", dataFile)
				}
			}
		}
	},
}

func init() {
	exportCmd.Flags().StringP("table", "T", "", "Table name to export (if not set, exports all tables)")
	exportCmd.Flags().StringP("output", "o", "exported", "Output directory for exported files")
	exportCmd.Flags().StringP("json", "j", "", "Specify json file  name to load (e.g., dsn.json,)")
	exportCmd.Flags().Bool("schema-only", false, "Export only schema")
	exportCmd.Flags().Bool("data-only", false, "Export only data")

	// Add the export command to your root command or application
	goFrame.AddCommand(exportCmd)
}
