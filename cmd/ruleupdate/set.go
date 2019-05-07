// Copyright 2019 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// in the LICENSE file.

package main

type empty struct{}

type stringSet struct {
	m map[string]empty
}

func (s *stringSet) insert(key string) {
	if s.m == nil {
		s.m = make(map[string]empty)
	}
	s.m[key] = empty{}
}

func (s *stringSet) contains(key string) bool {
	_, ok := s.m[key]
	return ok
}
