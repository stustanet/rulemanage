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

func (db *database) currentRules(handleRule handleRuleFunc) error {
	rows, err := db.db.Query("SELECT sid, rev, deactivated_at, deleted_at FROM rule ORDER BY sid ASC")
	if err != nil {
		return err
	}

	for rows.Next() {
		var (
			sid  int
			meta ruleMeta
		)
		if err := rows.Scan(&sid, &meta.rev, &meta.deactivatedAt, &meta.deletedAt); err != nil {
			return err
		}
		handleRule(sid, meta)
	}
	return rows.Err()
}

func (db *database) deleteRule(sid int) (err error) {
	_, err = db.db.Exec(`UPDATE rule SET deleted_at = NOW() WHERE sid = $1 AND deleted_at IS NULL`, sid)
	return
}

func (db *database) saveRule(sid int, newRev int16, line string, file string, update bool) (err error) {
	if update {
		_, err = db.db.Exec(`UPDATE rule SET rev = $1, file = $2, pattern = $3, updated_at = NOW(), deleted_at = NULL WHERE sid = $4`, newRev, file, line, sid)
		return
	} else {
		_, err = db.db.Exec(`INSERT INTO rule(sid, rev, file, pattern) VALUES ($1, $2, $3, $4)`, sid, newRev, file, line)
		return
	}
}
