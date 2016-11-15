package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	db *sql.DB
}

func initDB() (DataBase DB, err error) {
	DataBase.db, err = sql.Open("sqlite3", "./ytsearch.db")

	if err != nil {
		return DataBase, err
	}

	sqlStmt := `
		create table channel (
				id integer not null primary key,
        title text,
        channel_id text,
        description text,
        thumb_url text
		);
		`
	_, _ = DataBase.db.Exec(sqlStmt)

	return DataBase, nil
}

func (DataBase *DB) Insert(channelID, title, description, thumbURL string) (id int64, err error) {
	tx, err := DataBase.db.Begin()
	if err != nil {
		return id, err
	}

	stmt, err := tx.Prepare("insert into channel(channel_id, title, description, thumb_url) values (?, ?, ?, ?)")
	if err != nil {
		return id, err
	}
	defer stmt.Close()
	result, err := stmt.Exec(channelID, title, description, thumbURL)
	if err != nil {
		return id, err
	}
	tx.Commit()

	id, _ = result.LastInsertId()

	return id, nil
}

type Rows struct {
	ID          int
	ChannelID   string
	Title       string
	Description string
	ThumbURL    string
}

func (DataBase *DB) Select() (selectRows []Rows, err error) {
	rows, err := DataBase.db.Query("select id, channel_id, title, description, thumb_url from channel")
	if err != nil {
		return selectRows, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var channel_id, title, description, thumb_url string
		err = rows.Scan(&id, &channel_id, &title, &description, &thumb_url)

		selectRows = append(selectRows, Rows{id, channel_id, title, description, thumb_url})
	}

	return selectRows, nil
}

func (DataBase *DB) Delete(channelID string) (err error) {
	_, err = DataBase.db.Exec("delete from channel where channel_id = ?", channelID)
	if err != nil {
		return err
	}
	return nil
}
