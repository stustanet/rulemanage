// Copyright 2019 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// in the LICENSE file.

package main

const (
	stateInserting = iota
	stateFinding
	stateDeleting
)

type ruleIndex struct {
	pos   int
	sids  []int
	meta  []ruleMeta
	seen  []bool
	state byte
}

type ruleMeta struct {
	rev    int16
	active bool
}

// insert inserts a rule's sid and meta data into the index.
func (ri *ruleIndex) insert(sid int, rev int16, active bool) {
	//fmt.Println(sid, rev, active)

	// common case: rules are inserted in ascending order
	if len(ri.sids) == 0 || ri.sids[len(ri.sids)-1] < sid {
		ri.sids = append(ri.sids, sid)
		ri.meta = append(ri.meta, ruleMeta{rev, active})
		ri.seen = append(ri.seen, false)
		return
	}
	panic("TBD")
}

// find attempts to find a given sid in the index.
// If it could be found, it is marked as seen and its meta data returned.
func (ri *ruleIndex) find(sid int) (found bool, rev int16, active bool) {
	n := len(ri.sids) - 1
	if n < 0 {
		return
	}

	pos := ri.pos

	// binary search if the gap is too big
	for min, max := 0, n; max-min > 10; {
		// figure out in which direction we need to search
		if diff := sid - ri.sids[pos]; diff > 10 {
			min = pos
		} else if diff < -10 {
			max = pos
		} else {
			// stop binary search early if the difference is small
			break
		}
		pos = (min + max) / 2
	}

	// search in ascending order
	for pos < n && sid > ri.sids[pos] {
		pos++
	}

	// search in descending order
	for pos > 0 && sid < ri.sids[pos] {
		pos--
	}

	if ri.sids[pos] == sid {
		ri.seen[pos] = true
		ri.pos = pos
		meta := ri.meta[pos]
		return true, meta.rev, meta.active
	}

	return false, 0, false
}

// nextUnseen returns the next SID that wasn't seen or -1 if there is none
func (ri *ruleIndex) nextUnseen() (pos int) {
	pos = ri.pos
	if ri.state != stateDeleting {
		pos = len(ri.seen)
		ri.state = stateDeleting
	}
	for {
		pos--
		if pos < 0 {
			ri.pos = -1
			return -1
		}
		if !ri.seen[pos] {
			ri.pos = pos
			return ri.sids[pos]
		}
	}
}
