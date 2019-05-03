// Copyright 2019 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// in the LICENSE file.

package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"database/sql"
	"flag"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/gonids"

	_ "github.com/lib/pq"
)

type empty struct{}

var db *sql.DB

var knownRules ruleIndex

func main() {
	cfgPath := flag.String("cfg", "/etc/rulemanage.toml", "path to config file")
	flag.Parse()

	var cfg Config
	cfg.loadFile(*cfgPath)

	var err error
	db, err = sql.Open("postgres", cfg.Database.DSN)
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	getCurrentRules()

	for name, src := range cfg.Rules {
		handleSource(name, src)
	}

	findDeletedRules()
}

func getCurrentRules() {
	rows, err := db.Query("SELECT sid, rev, active FROM rule ORDER BY sid ASC")
	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		var (
			sid    int
			rev    int16
			active bool
		)
		if err := rows.Scan(&sid, &rev, &active); err != nil {
			log.Fatal(err)
		}
		knownRules.insert(sid, rev, active)
	}
}

func findDeletedRules() {
	for {
		sid := knownRules.nextUnseen()
		if sid < 0 {
			return
		}
		log.Println("SID", sid, "was deleted")
		_, err := db.Exec(`DELETE FROM rule WHERE sid = $1`, sid)
		if err != nil {
			log.Println("[ERROR]", err, "for SID", sid)
		}
	}
}

func handleSource(name string, src source) {
	log.Printf("fetching source '%s' from %s\n", name, src.Source)

	wantedFiles := make(map[string]empty)
	for _, path := range src.Files {
		wantedFiles[path] = empty{}
	}

	var httpClient = &http.Client{
		Timeout: time.Second * 60,
	}

	resp, err := httpClient.Get(src.Source)
	if err != nil {
		log.Fatal(err)
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// open and iterate through the files in the archive.
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // end of archive
		}
		if err != nil {
			log.Fatal(err)
		}

		// ignore everything that is not a regular file
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		if !strings.HasSuffix(hdr.Name, ".rules") {
			continue
		}

		if _, ok := wantedFiles[hdr.Name]; !ok {
			log.Println("skipping", hdr.Name)
			continue
		}

		log.Printf("parsing %s\n", hdr.Name)
		scanner := bufio.NewScanner(tr)
		for scanner.Scan() {
			l := scanner.Text()
			if len(l) == 0 || l[0] == '#' {
				continue
			}
			rule := parseRule(l)
			if rule == nil {
				continue
			}
			saveRule(l, name+":"+hdr.Name, rule)
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}
}

func parseRule(line string) *gonids.Rule {
	r, err := gonids.ParseRule(line)
	if err != nil {
		log.Println("[ERROR]", err, "in rule", line)
		return nil
	}

	// ignore disabled/commented rules
	if r.Disabled {
		return nil
	}

	// we only care about 'alert' rules
	if r.Action != "alert" {
		return nil
	}

	if r.SID == 0 {
		log.Println("[ERROR] 0 SID in rule", line)
		return nil
	}

	return r
}

func saveRule(line string, file string, r *gonids.Rule) bool {
	found, rev, _ := knownRules.find(r.SID)
	if !found {
		log.Println("SID", r.SID, "is new")
		_, err := db.Exec(`INSERT INTO rule(sid, rev, file, pattern, active) VALUES ($1, $2, $3, $4, TRUE)`, r.SID, r.Revision, file, line)
		if err != nil {
			log.Println("[ERROR]", err, "for SID", r.SID)
		}
	} else if r.Revision > int(rev) {
		log.Println("SID", r.SID, "was updated")
		_, err := db.Exec(`UPDATE rule SET rev = $1, file = $2, pattern = $3, updated_at = NOW() WHERE sid = $4`, r.Revision, file, line, r.SID)
		if err != nil {
			log.Println("[ERROR]", err, "for SID", r.SID)
		}
	}

	return false
}
