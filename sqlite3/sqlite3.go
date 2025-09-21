package sqlite3

import (
	"astera"
	"database/sql"
	"embed"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

var (
	//go:embed migrations/*.sql
	fs embed.FS
)

type DB struct {
	db *sql.DB
}

func NewDB(database string) (*DB, error) {
	d, err := iofs.New(fs, "migrations")
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", database+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	migrator, err := migrate.NewWithSourceInstance("iofs", d, "sqlite3://"+database)
	if err != nil {
		return nil, err
	}

	err = migrator.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, err
	}

	return &DB{db: db}, nil
}

// On conflict it does nothing but we should check the hash for example and report if it is different
func (d *DB) InsertModule(module *astera.Module) error {
	query := `INSERT INTO module (name, version, mod, info, zip_hash, zip) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT DO NOTHING;`

	_, err := d.db.Exec(query,
		module.Name,
		module.Version,
		module.Mod,
		module.Info,
		module.ZipHash,
		module.Zip)
	if err != nil {
		return err
	}

	return nil
}

func (d *DB) GetVersionList(name string) ([]string, error) {
	query := `SELECT version FROM module WHERE name = ?`
	rows, err := d.db.Query(query, name)
	if err != nil {
		return nil, err
	}

	versions := make([]string, 0)
	for rows.Next() {
		var version string
		err = rows.Scan(&version)
		if err != nil {
			break
		}

		versions = append(versions, version)
	}

	if closeErr := rows.Close(); closeErr != nil {
		return nil, closeErr
	}

	if err != nil {
		return nil, err
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return versions, nil
}

func (d *DB) GetVersionInfo(name, version string) ([]byte, error) {
	query := `SELECT info FROM module WHERE name = ? AND version = ?`
	row := d.db.QueryRow(query, name, version)

	var info sql.Null[[]byte]
	err := row.Scan(&info)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, astera.ErrModuleNotFound
		}

		return nil, err
	}

	if !info.Valid {
		return nil, astera.ErrModuleNotFound
	}

	return info.V, nil
}

func (d *DB) GetModFile(name, version string) ([]byte, error) {
	query := `SELECT mod FROM module WHERE name = ? AND version = ?`
	row := d.db.QueryRow(query, name, version)

	var mod sql.Null[[]byte]
	err := row.Scan(&mod)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, astera.ErrModuleNotFound
		}

		return nil, err
	}

	if !mod.Valid {
		return nil, astera.ErrModuleNotFound
	}

	return mod.V, nil
}

func (d *DB) GetModuleZip(name, version string) ([]byte, error) {
	query := `SELECT zip FROM module WHERE name = ? AND version = ?`
	row := d.db.QueryRow(query, name, version)
	var zip []byte

	err := row.Scan(&zip)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, astera.ErrModuleNotFound
		}

		return nil, err
	}

	return zip, nil
}

// IsModule check the present of the module in the database
func (d *DB) ModuleExists(name string, version string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM module WHERE name = ? AND version = ?)`
	var exists bool
	err := d.db.QueryRow(query, name, version).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil

}
