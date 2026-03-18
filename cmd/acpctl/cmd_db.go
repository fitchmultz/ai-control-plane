// cmd_db.go - Database command tree assembly.
//
// Purpose:
//   - Compose the typed `db` command tree from focused backup, retention,
//     restore, and verification command handlers.
//
// Responsibilities:
//   - Define the root `db` command surface and shared option payloads.
//
// Scope:
//   - Database command metadata only.
//
// Usage:
//   - Invoked through `acpctl db <subcommand>`.
//
// Invariants/Assumptions:
//   - Database admin logic remains in typed helpers and internal services.
package main

type dbBackupOptions struct {
	BackupName string
}

type dbRestoreOptions struct {
	BackupFile string
}

func dbCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "db",
		Summary:     "Database backup, restore, and inspection operations",
		Description: "Database backup, restore, and inspection operations.",
		Examples: []string{
			"acpctl db status",
			"acpctl db backup",
			"acpctl db backup-retention --check",
			"acpctl db off-host-drill --manifest demo/logs/recovery-inputs/off_host_recovery.yaml",
			"acpctl db dr-drill",
		},
		Children: []*commandSpec{
			newNativeLeafCommandSpec("status", "Show database status and statistics", runDBStatus),
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "backup",
				Summary:     "Create database backup",
				Description: "Backup the PostgreSQL database to a timestamped compressed file.",
				Arguments: []commandArgumentSpec{
					{Name: "backup-name", Summary: "Optional custom backup name"},
				},
				Bind: bindParsedValue(func(input parsedCommandInput) dbBackupOptions {
					return dbBackupOptions{BackupName: input.NormalizedArgument(0)}
				}),
				Run: runDBBackup,
			}),
			dbBackupRetentionCommandSpec(),
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "restore",
				Summary:     "Restore embedded database from backup",
				Description: "Restore the PostgreSQL database from a backup file.",
				Arguments: []commandArgumentSpec{
					{Name: "backup-file", Summary: "Optional backup file path"},
				},
				Bind: bindParsedValue(func(input parsedCommandInput) dbRestoreOptions {
					return dbRestoreOptions{BackupFile: input.NormalizedArgument(0)}
				}),
				Run: runDBRestore,
			}),
			dbOffHostDrillCommandSpec(),
			newNativeLeafCommandSpec("shell", "Open database shell", runDBShell),
			newNativeLeafCommandSpec("dr-drill", "Create a fresh backup and verify restore into a scratch database", runDBDRDrillTyped),
		},
	}
}
