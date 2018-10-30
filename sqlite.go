package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

type sqlitedb struct {
	Path string
	db *sql.DB
}

func (db *sqlitedb) AccountCreate(user, password string) (error) {
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	query, err := tx.Prepare("INSERT INTO accounts (user, password) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer query.Close()
	if _, err := query.Exec(user, password); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *sqlitedb) SessionCreate(user, password string) (string, error) {
	tx, err := db.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	query, err := tx.Prepare("SELECT user FROM accounts WHERE user = ?")
	if err != nil {
		return "", err
	}
	var founduser string
	if err := query.QueryRow(user).Scan(&founduser); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("Invalid user or password")
		}
		return "", err
	}
	query, err = tx.Prepare("DELETE FROM sessions WHERE user = ?")
	if err != nil {
		return "", err
	}
	if _, err := query.Exec(user); err != nil {
		return "", err
	}
	query, err = tx.Prepare("INSERT INTO sessions (id, user) VALUES (?, ?)")
	if err != nil {
		return "", err
	}
	defer query.Close()
	cookie := RandomCookie()
	if _, err := query.Exec(cookie, user); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}

	return cookie, nil
}

func (db *sqlitedb) SessionGet(cookie string) (string, error) {
	query, err := db.db.Prepare("SELECT user FROM sessions WHERE id = ?")
	if err != nil {
		return "", err
	}
	var user string
	if err := query.QueryRow(cookie).Scan(&user); err != nil {
		return "", err
	}
	return user, nil
}

func migrate(db *sql.DB) error {
	sql := "create table accounts (user text not null primary key, password text not null);"
	_, err := db.Exec(sql)
	if err != nil {
		return err
	}

	sql = "create table sessions (id text not null, user text not null);"
	_, err = db.Exec(sql)
	if err != nil {
		return err
	}

	sql = `create table favorites 
		(id text not null,
		 user text not null,
		 embed_url text not null,
		 still_url text not null,
		 downsized_url text not null);`
	_, err = db.Exec(sql)
	if err != nil {
		return err
	}

	sql = `create table tags
		(tag text not null,
		 user text not null,
		 favorite text not null,
		 foreign key(favorite) references favorites(id));`
	_, err = db.Exec(sql)
	if err != nil {
		return err
	}
	return nil
}

func NewSqliteDB(path string) (Db, error) {
	fmt.Printf("NewSqliteDB\n")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	var name string
	if err := db.QueryRow("SELECT name FROM sqlite_master WHERE name = 'accounts'").Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			err = migrate(db)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return &sqlitedb{
		Path: path,
		db: db,
	}, nil
}
