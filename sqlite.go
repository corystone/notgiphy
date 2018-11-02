package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

type sqlitedb struct {
	Path string
	db   *sql.DB
}

func (db *sqlitedb) AccountCreate(user, password string) error {
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

func (db *sqlitedb) TagCreate(tag Tag, user string) error {
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	query, err := tx.Prepare("INSERT INTO tags (favorite, tag, user) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer query.Close()
	if _, err := query.Exec(tag.Favorite, tag.Tag, user); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *sqlitedb) TagDelete(tag Tag, user string) error {
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	query, err := tx.Prepare("DELETE FROM tags WHERE favorite = ? and tag = ? and user = ?")
	if err != nil {
		return err
	}
	defer query.Close()
	if _, err := query.Exec(tag.Favorite, tag.Tag, user); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *sqlitedb) tagListQuery(favorite, user string) ([]Tag, error) {
	records := []Tag{}
	var rows *sql.Rows
	var err error
	if favorite == "" {
		rows, err = db.db.Query(`SELECT tag, favorite
                                FROM tags
                                WHERE user = ?
                                GROUP BY tag
				ORDER BY tag`, user)
	} else {
		rows, err = db.db.Query(`SELECT tag, favorite
                                FROM tags
                                WHERE user = ? and favorite = ?
                                ORDER BY tag`, user, favorite)
	}
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.Tag, &tag.Favorite); err != nil {
			return nil, err
		}
		records = append(records, tag)
	}
	rows.Close()
	return records, nil
}

func (db *sqlitedb) TagList(user string) ([]Tag, error) {
	return db.tagListQuery("", user)
}

func (db *sqlitedb) FavoriteTagList(favorite, user string) ([]Tag, error) {
	return db.tagListQuery(favorite, user)
}

func (db *sqlitedb) FavoriteCreate(gif *Gif, user string) error {
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	query, err := tx.Prepare("INSERT INTO favorites (id, user, url, still_url, downsized_url) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer query.Close()
	if _, err := query.Exec(gif.Id, user, gif.URL, gif.StillURL, gif.DownsizedURL); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *sqlitedb) FavoriteDelete(id, user string) error {
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	query, err := tx.Prepare("DELETE FROM tags WHERE favorite = ? and user = ?")
	if err != nil {
		return err
	}
	defer query.Close()
	if _, err := query.Exec(id, user); err != nil {
		return err
	}
	query, err = tx.Prepare("DELETE FROM favorites WHERE id = ? and user = ?")
	if err != nil {
		return err
	}
	defer query.Close()
	if _, err := query.Exec(id, user); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *sqlitedb) FavoriteGet(id, user string) (*Gif, error) {
	row := db.db.QueryRow(`SELECT id, url, still_url, downsized_url
                               FROM favorites
                               WHERE id = ? and user = ?`, id, user)
	gif := &Gif{}
	if err := row.Scan(&gif.Id, &gif.URL, &gif.StillURL, &gif.DownsizedURL); err != nil {
		return nil, err
	}
	return gif, nil
}

func (db *sqlitedb) favoriteListQuery(tag, user string) ([]Gif, error) {
	records := []Gif{}
	var rows *sql.Rows
	var err error
	if tag == "" {
		rows, err = db.db.Query(`SELECT id, url, still_url, downsized_url
					FROM favorites
					WHERE user = ?
					ORDER BY id`, user)
	} else {
		rows, err = db.db.Query(`SELECT f.id, f.url, f.still_url, f.downsized_url
					FROM favorites f
					INNER JOIN tags t ON f.id = t.favorite
							  AND f.user = t.user
					WHERE f.user = ? and t.tag = ?
					ORDER BY id`, user, tag)
	}
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var gif Gif
		if err := rows.Scan(&gif.Id, &gif.URL, &gif.StillURL, &gif.DownsizedURL); err != nil {
			return nil, err
		}
		records = append(records, gif)
	}
	rows.Close()
	return records, nil
}

func (db *sqlitedb) FavoriteListByTag(tag, user string) ([]Gif, error) {
	return db.favoriteListQuery(tag, user)
}

func (db *sqlitedb) FavoriteList(user string) ([]Gif, error) {
	return db.favoriteListQuery("", user)
}

func migrate(db *sql.DB) error {
	fmt.Printf("Creating database schema\n")
	sql := "create table accounts (user text not null primary key, password text not null);"
	_, err := db.Exec(sql)
	if err != nil {
		return err
	}

	sql = `create table sessions
		(id text primary key not null,
		 user text not null,
		 foreign key(user) references accounts(user));`
	_, err = db.Exec(sql)
	if err != nil {
		return err
	}

	sql = `create table favorites 
		(id text not null,
		 user text not null,
		 url text not null,
		 still_url text not null,
		 downsized_url text not null,
		 primary key (id, user),
		 foreign key(user) references accounts(user));`
	_, err = db.Exec(sql)
	if err != nil {
		return err
	}

	sql = `create table tags
		(tag text not null,
		 user text not null,
		 favorite text not null,
		 primary key (tag, user, favorite),
		 foreign key(user) references accounts(user),
		 foreign key(favorite, user) references favorites(id, user));`
	_, err = db.Exec(sql)
	if err != nil {
		return err
	}
	return nil
}

func NewSqliteDB(path string) (Db, error) {
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=1")
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
		db:   db,
	}, nil
}
