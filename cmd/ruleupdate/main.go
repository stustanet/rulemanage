// Copyright 2019 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// in the LICENSE file.

package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"flag"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/gonids"

	"github.com/lib/pq"
)

type ruleMeta struct {
	rev           int16
	deactivatedAt pq.NullTime
	deletedAt     pq.NullTime
}

type handleRuleFunc func(sid int, meta ruleMeta)

type saveRuleFunc func(sid int, newRev int16, line string, file string, update bool) error

type empty struct{}

var knownRules ruleIndex

func main() {
	cfgPath := flag.String("cfg", "/etc/rulemanage.toml", "path to config file")
	flag.Parse()

	var cfg Config
	cfg.loadFile(*cfgPath)

	var db database
	err := db.connect(cfg.Database.DSN)
	if err != nil {
		log.Fatal(err)
	}

	err = db.currentRules(knownRules.insert)
	if err != nil {
		log.Fatal(err)
	}

	for name, src := range cfg.Rules {
		handleSource(db.saveRule, name, src)
	}

	for {
		sid := knownRules.nextUnseen()
		if sid < 0 {
			return
		}
		log.Println("SID", sid, "was deleted")
		if err := db.deleteRule(sid); err != nil {
			log.Println("[ERROR]", err, "for SID", sid)
		}
	}
}

func handleSource(save saveRuleFunc, name string, src source) {
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

			fileID := name + ":" + hdr.Name
			sid := rule.SID
			rev := int16(rule.Revision)
			found, meta := knownRules.find(sid)
			if !found {
				log.Println("SID", sid, "is new")
			} else if rev > meta.rev || meta.deletedAt.Valid {
				log.Println("SID", sid, "was updated")
			}
			if err := save(sid, rev, l, fileID, found); err != nil {
				log.Println("[ERROR]", err, "for SID", sid)
			}

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
