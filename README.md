# SQL Migration CLI Tool

A simple and flexible command-line tool for exporting and importing SQL table schemas and data between databases using `.sql` files. This tool supports **MySQL**, **PostgreSQL**, and **SQLite**.

---
## Features

- ✅ Export table **schemas** and/or **data** to `.sql` files.
- 📥 Import table **schemas** and/or **data** from `.sql` files.
- 🔁 Supports **MySQL**, **PostgreSQL**, and **SQLite**.
- 🔍 Select a specific table or operate on **all tables**.
- ⚙️ Load database settings from a JSON config file (`dsn.json`).
---

## Installation

Make sure you have Go installed. Then, build the tool:

```bash
go install  github.com/semay-cli/sql-migration@latest
```

##  Sample JSON File

```json
{
  "Driver": "mysql",
  "ExportDatabaseDSN": "user:password@tcp(127.0.0.1:3306)/source_db",
  "ImportDatabaseDSN": "user:password@tcp(127.0.0.1:3306)/target_db"
}
```
## Available Flags

| Flag             | Type   | Description                                                                           |
| ---------------- | ------ | ------------------------------------------------------------------------------------- |
| `-T`, `--table`  | string | Name of a specific table to export. If omitted, **all tables** are exported.          |
| `-i, --input`   | Input directory for `.sql` files (default: `exported`)                                          |
| `-o`, `--output` | string | Output directory where `.sql` files are saved. Default is `exported`.                 |
| `-j`, `--json`   | string | Name of the config JSON file to use (e.g., `dev`, `staging`). Defaults to `dsn.json`. |
| `--schema-only`  | bool   | Export **only** the schema (`CREATE TABLE` statements).                               |
| `--data-only`    | bool   | Export **only** the data (`INSERT INTO` statements).                                  |



##  Example

```bash
sql-migration import --table orders --data-only
```