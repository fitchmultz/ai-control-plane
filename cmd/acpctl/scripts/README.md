# Scripts Package

This package contains Go implementations of former shell scripts.
Each script that was migrated from shell to Go has a corresponding file here.

## Migration Pattern

1. Original: demo/scripts/script_name.sh
2. Go: cmd/acpctl/scripts/script_name.go
3. Registration: cmd_delegated.go or main.go
4. Makefile: Update to use acpctl command
5. Delete: Original shell script

## Migrated Scripts

- health_check.sh → health package (internal/health/)
- db_backup.sh → cmd_db_ops.go
- db_restore.sh → cmd_db_ops.go
