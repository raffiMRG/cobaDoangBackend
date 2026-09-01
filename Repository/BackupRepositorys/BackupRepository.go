package BackupRepositorys

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"web_backend/Model/Bookmark"
	"web_backend/Model/NewFolder"
	tbFolder "web_backend/Model/TbFolder"
	"web_backend/Model/User"
)

var createTableStatements = map[string]string{
	"users": "CREATE TABLE IF NOT EXISTS `users` (\n" +
		"  `id` INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,\n" +
		"  `username` VARCHAR(255) NOT NULL UNIQUE,\n" +
		"  `password` VARCHAR(255) NOT NULL,\n" +
		"  `create_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP\n" +
		");",
	// refresh_tokens holds live login sessions, not backup data — its rows
	// are intentionally never exported/imported — but its table structure
	// still has to be dropped/recreated in "full" mode, since it FKs to
	// `users` and would otherwise block DROP TABLE `users`.
	"refresh_tokens": "CREATE TABLE IF NOT EXISTS `refresh_tokens` (\n" +
		"  `id` INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,\n" +
		"  `user_id` INT UNSIGNED NOT NULL,\n" +
		"  `token_hash` VARCHAR(255) NOT NULL,\n" +
		"  `expires_at` TIMESTAMP NOT NULL,\n" +
		"  `revoked` BOOLEAN NOT NULL DEFAULT FALSE,\n" +
		"  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,\n" +
		"  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE\n" +
		");",
	"folders": "CREATE TABLE IF NOT EXISTS `folders` (\n" +
		"  `id` INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,\n" +
		"  `name` VARCHAR(255),\n" +
		"  `thumbnail` VARCHAR(500) NULL\n" +
		");",
	"new_folders": "CREATE TABLE IF NOT EXISTS `new_folders` (\n" +
		"  `id` INT AUTO_INCREMENT PRIMARY KEY,\n" +
		"  `name` VARCHAR(255) NOT NULL,\n" +
		"  `thumbnail` VARCHAR(500) NULL,\n" +
		"  `is_completed` BOOLEAN,\n" +
		"  `create_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP\n" +
		");",
	"bookmarks": "CREATE TABLE IF NOT EXISTS `bookmarks` (\n" +
		"  `id` INT NOT NULL AUTO_INCREMENT,\n" +
		"  `folder_id` INT NOT NULL,\n" +
		"  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,\n" +
		"  PRIMARY KEY (`id`),\n" +
		"  FOREIGN KEY (`folder_id`) REFERENCES `new_folders`(`id`) ON DELETE CASCADE\n" +
		");",
}

// structuralTablesInDependencyOrder lists every table whose schema is
// touched by "full" mode DROP/CREATE, parent-first (refresh_tokens and
// bookmarks FK to users and new_folders respectively).
var structuralTablesInDependencyOrder = []string{"users", "refresh_tokens", "folders", "new_folders", "bookmarks"}

// dataTablesInDependencyOrder lists tables whose row *data* is actually
// exported/imported — refresh_tokens is deliberately excluded (live
// sessions, not backup data).
var dataTablesInDependencyOrder = []string{"users", "folders", "new_folders", "bookmarks"}

func reversed(items []string) []string {
	out := make([]string, len(items))
	for i, v := range items {
		out[len(items)-1-i] = v
	}
	return out
}

func sqlQuote(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`'`, `\'`,
		"\x00", `\0`,
		"\n", `\n`,
		"\r", `\r`,
		"\x1a", `\Z`,
	)
	return "'" + replacer.Replace(s) + "'"
}

func sqlValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return sqlQuote(val)
	case bool:
		if val {
			return "1"
		}
		return "0"
	case time.Time:
		return sqlQuote(val.Format("2006-01-02 15:04:05"))
	default:
		return fmt.Sprintf("%v", val)
	}
}

// ExportDatabase dumps users, folders, new_folders and bookmarks as a
// self-contained .sql script. mode "full" includes DROP/CREATE TABLE
// statements (restorable to an empty database); any other mode ("data")
// only clears and re-inserts rows into already-existing tables.
func ExportDatabase(db *gorm.DB, mode string) (string, error) {
	var sb strings.Builder

	if mode == "full" {
		for _, table := range reversed(structuralTablesInDependencyOrder) {
			sb.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS `%s`;\n", table))
		}
		for _, table := range structuralTablesInDependencyOrder {
			sb.WriteString(createTableStatements[table])
			sb.WriteString("\n")
		}
	} else {
		for _, table := range reversed(dataTablesInDependencyOrder) {
			sb.WriteString(fmt.Sprintf("DELETE FROM `%s`;\n", table))
		}
	}

	if err := exportUsers(db, &sb); err != nil {
		return "", err
	}
	if err := exportFolders(db, &sb); err != nil {
		return "", err
	}
	if err := exportNewFolders(db, &sb); err != nil {
		return "", err
	}
	if err := exportBookmarks(db, &sb); err != nil {
		return "", err
	}

	return sb.String(), nil
}

func exportUsers(db *gorm.DB, sb *strings.Builder) error {
	var rows []User.User
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		sb.WriteString(fmt.Sprintf(
			"INSERT INTO `users` (`id`, `username`, `password`, `create_at`) VALUES (%s, %s, %s, %s);\n",
			sqlValue(r.ID), sqlValue(r.Username), sqlValue(r.Password), sqlValue(r.CreateAt),
		))
	}
	return nil
}

func exportFolders(db *gorm.DB, sb *strings.Builder) error {
	var rows []tbFolder.Folder
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		sb.WriteString(fmt.Sprintf(
			"INSERT INTO `folders` (`id`, `name`, `thumbnail`) VALUES (%s, %s, %s);\n",
			sqlValue(r.ID), sqlValue(r.Name), sqlValue(r.Thumbnail),
		))
	}
	return nil
}

func exportNewFolders(db *gorm.DB, sb *strings.Builder) error {
	var rows []NewFolder.NewFolder
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		sb.WriteString(fmt.Sprintf(
			"INSERT INTO `new_folders` (`id`, `name`, `thumbnail`, `is_completed`, `create_at`) VALUES (%s, %s, %s, %s, %s);\n",
			sqlValue(r.ID), sqlValue(r.Name), sqlValue(r.Thumbnail), sqlValue(r.IsCompleted), sqlValue(r.CreateAt),
		))
	}
	return nil
}

func exportBookmarks(db *gorm.DB, sb *strings.Builder) error {
	var rows []Bookmark.Bookmark
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		sb.WriteString(fmt.Sprintf(
			"INSERT INTO `bookmarks` (`id`, `folder_id`, `created_at`) VALUES (%s, %s, %s);\n",
			sqlValue(r.ID), sqlValue(r.FolderID), sqlValue(r.CreatedAt),
		))
	}
	return nil
}

// ImportDatabase executes a previously exported .sql script, wrapped in a
// transaction so a failed INSERT rolls back the data written so far in
// "data" mode. Note MySQL implicitly commits DDL (DROP/CREATE TABLE)
// immediately regardless of transaction state, so "full" mode is not
// purely atomic — a mid-script failure there can leave freshly (re)created,
// empty tables. Every DROP/CREATE uses IF EXISTS/IF NOT EXISTS, so a
// straightforward re-run after fixing the cause converges cleanly. Only
// trusted for content produced by ExportDatabase (statements are split on
// ";\n", which never occurs inside an exported string value since newlines
// in source data are escaped to the two-character sequence \n).
func ImportDatabase(db *gorm.DB, sqlContent string) error {
	statements := strings.Split(sqlContent, ";\n")

	err := db.Transaction(func(tx *gorm.DB) error {
		for _, stmt := range statements {
			trimmed := strings.TrimSpace(stmt)
			if trimmed == "" {
				continue
			}
			// LOCK TABLES is session-scoped in MySQL: if a foreign dump
			// (e.g. a raw mysqldump, not our own ExportDatabase output)
			// locks tables and then a later statement fails, the pooled
			// connection is left locked and every subsequent query on it
			// errors with "table was not locked" until the connection is
			// discarded — effectively breaking the whole app. We never
			// need table locking for this import model, so drop it.
			upper := strings.ToUpper(trimmed)
			if strings.HasPrefix(upper, "LOCK TABLES") || strings.HasPrefix(upper, "UNLOCK TABLES") {
				continue
			}
			if err := tx.Exec(trimmed).Error; err != nil {
				return fmt.Errorf("failed executing statement %q: %w", trimmed, err)
			}
		}
		return nil
	})

	// Safety net: if anything on this connection ended up locked despite
	// the filtering above, release it before returning so the connection
	// is safe to reuse from the pool.
	db.Exec("UNLOCK TABLES")

	return err
}
