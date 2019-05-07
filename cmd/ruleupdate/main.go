// Copyright 2019 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// in the LICENSE file.

package main

import (
	"flag"
	"log"

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

func handleRule(saveRule saveRuleFunc, fileID, line string) {
	if len(line) == 0 || line[0] == '#' {
		return
	}
	rule := parseRule(line)
	if rule == nil {
		return
	}

	sid := rule.SID
	rev := int16(rule.Revision)
	found, meta := knownRules.find(sid)
	if !found {
		log.Println("SID", sid, "is new")
	} else if rev > meta.rev || meta.deletedAt.Valid {
		log.Println("SID", sid, "was updated")
	}
	if err := saveRule(sid, rev, line, fileID, found); err != nil {
		log.Println("[ERROR]", err, "for SID", sid)
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
