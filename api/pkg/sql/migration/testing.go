package lmigration

func NewMigrationSet(set MigrationSet) map[string]MigrationSet {
	return map[string]MigrationSet{"sqlite": set}
}
