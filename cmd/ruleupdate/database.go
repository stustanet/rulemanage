// Copyright 2019 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// in the LICENSE file.

package main

import (
	"database/sql"

	_ "github.com/lib/pq"
)

type database struct {
	db *sql.DB
}

func (db *database) connect(dsn string) (err error) {
	db.db, err = sql.Open("postgres", dsn)
	if err != nil {
		return
	}

	return db.db.Ping()
}

func (db *database) currentRules(handle handleRuleFunc) error {
	rows, err := db.db.Query("SELECT sid, rev, active FROM rule ORDER BY sid ASC")
	if err != nil {
		return err
	}

	for rows.Next() {
		var (
			sid    int
			rev    int16
			active bool
		)
		if err := rows.Scan(&sid, &rev, &active); err != nil {
			return err
		}
		handle(sid, rev, active)
	}
	return rows.Err()
}

func (db *database) deleteRule(sid int) (err error) {
	_, err = db.db.Exec(`DELETE FROM rule WHERE sid = $1`, sid)
	return
}

func (db *database) saveRule(sid int, newRev int16, line string, file string, update bool) (err error) {
	if update {
		_, err = db.db.Exec(`UPDATE rule SET rev = $1, file = $2, pattern = $3, updated_at = NOW() WHERE sid = $4`, newRev, file, line, sid)
		return
	} else {
		_, err = db.db.Exec(`INSERT INTO rule(sid, rev, file, pattern, active) VALUES ($1, $2, $3, $4, TRUE)`, sid, newRev, file, line)
		return
	}
}
