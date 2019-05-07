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
	"os"
	"path/filepath"
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
		handleLocalSource(saveRule, name, src, wantedFiles)
		return
	case strings.HasPrefix(src.Source, "https://"),
		strings.HasPrefix(src.Source, "http://"):
		handleHTTPSource(saveRule, name, src, wantedFiles)
	}
}

func handleLocalSource(saveRule saveRuleFunc, name string, src source, wantedFiles stringSet) {
	if err := filepath.Walk(src.Source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		relPath, err := filepath.Rel(src.Source, path)

		if !wantedFiles.contains(relPath) {
			log.Println("skipping", relPath)
		}
		if err != nil {
			return err
		}

		log.Printf("parsing %s\n", relPath)
		fileID := name + ":" + relPath

		file, err := os.Open(path)
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			handleRule(saveRule, fileID, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
		return file.Close()
	}); err != nil {
		log.Fatalf("error walking the path %q: %v\n", src.Source, err)
		return
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
