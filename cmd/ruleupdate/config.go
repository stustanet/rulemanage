// Copyright 2019 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// in the LICENSE file.

package main

import (
	"log"
	"os"

	"github.com/naoina/toml"
)

type Config struct {
	Database struct {
		DSN string
	}
	Rules map[string]source
}

type source struct {
	Source string
	Files  []string
}

func (cfg *Config) loadFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	if err = toml.NewDecoder(f).Decode(&cfg); err != nil {
		log.Fatal(err)
	}
	f.Close()
}
