package dbx

// pendingMigrations returns the set of versions not yet recorded in migrations_history.
//
// Upstream applies only versions above the newest recorded one, which silently skips an
// upstream migration dated before a fork migration. Versions below the first recorded one
// predate migrations_history (they were tracked in schema_migrations) and are never re-run.
func pendingMigrations(versions []int, lastVersion int) (map[int]bool, error) {
	rows, err := conn.Query("SELECT version FROM migrations_history")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	first := 0
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
		if first == 0 || version < first {
			first = version
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(applied) == 0 {
		first = lastVersion + 1
	}

	pending := make(map[int]bool)
	for _, version := range versions {
		if version >= first && !applied[version] {
			pending[version] = true
		}
	}
	return pending, nil
}
