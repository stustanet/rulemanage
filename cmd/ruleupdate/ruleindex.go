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

func (ri *ruleIndex) find(sid int) (found bool, rev int16, active bool) {
	n := len(ri.sids) - 1
	if n < 0 {
		return
	}

	pos := ri.pos

	// binary search if the gap is too big
	for diff, min, max := sid-ri.sids[pos], 0, n; (diff > 10 || diff < -10) && max-min > 10; diff = sid - ri.sids[pos] {
		if diff > 0 {
			min = pos
		} else {
			max = pos
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
		meta := ri.meta[pos]
		ri.pos = pos
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
