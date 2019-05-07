// Copyright 2019 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// in the LICENSE file.

package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func handleSource(saveRule saveRuleFunc, name string, src source) {
	log.Printf("fetching source '%s' from %s\n", name, src.Source)

	var wantedFiles stringSet
	for _, path := range src.Files {
		wantedFiles.insert(path)
	}

	switch {
	case strings.HasPrefix(src.Source, "/"):
		log.Println("LOCAL SOURCE")
		return
	case strings.HasPrefix(src.Source, "https://"),
		strings.HasPrefix(src.Source, "http://"):
		handleHTTPSource(saveRule, name, src, wantedFiles)
	}
}

func handleHTTPSource(saveRule saveRuleFunc, name string, src source, wantedFiles stringSet) {
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

		if !wantedFiles.contains(hdr.Name) {
			log.Println("skipping", hdr.Name)
			continue
		}

		log.Printf("parsing %s\n", hdr.Name)
		fileID := name + ":" + hdr.Name
		scanner := bufio.NewScanner(tr)
		for scanner.Scan() {
			handleRule(saveRule, fileID, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}
}
