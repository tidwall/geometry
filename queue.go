// Copyright 2021 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package geometry

// Priority Queue ordered by dist (smallest to largest)

type qkind int

const (
	qrect qkind = 1
	qseg  qkind = 2
)

type qnode struct {
	dist float64 // distance to rect or segment
	kind qkind   // rect or segment
	pos  int     // segment index or rect addr
	a, b Point   // segment or rect points
}

// cmp compares two nodes. Returns -1, 0, +1
func (qn qnode) cmp(other qnode) int {
	// The comparison priority of 'dist' then 'kind' then 'pos' is important
	// for stable ordering.
	var cmp int
	if qn.dist < other.dist {
		cmp = -1
	} else if qn.dist > other.dist {
		cmp = 1
	} else if qn.kind < other.kind {
		cmp = -1
	} else if qn.kind > other.kind {
		cmp = 1
	} else if qn.pos < other.pos {
		cmp = -1
	} else if qn.pos > other.pos {
		cmp = 1
	}
	return cmp
}

type queue []qnode

func (q *queue) push(node qnode) {
	*q = append(*q, node)
	nodes := *q
	i := len(nodes) - 1
	parent := (i - 1) / 2
	for ; i != 0 && nodes[parent].cmp(nodes[i]) > 0; parent = (i - 1) / 2 {
		nodes[parent], nodes[i] = nodes[i], nodes[parent]
		i = parent
	}
}

func (q *queue) pop() (qnode, bool) {
	nodes := *q
	if len(nodes) == 0 {
		return qnode{}, false
	}
	var n qnode
	n, nodes[0] = nodes[0], nodes[len(*q)-1]
	nodes = nodes[:len(nodes)-1]
	*q = nodes
	i := 0
	for {
		smallest := i
		left := i*2 + 1
		right := i*2 + 2
		if left < len(nodes) && nodes[left].cmp(nodes[smallest]) <= 0 {
			smallest = left
		}
		if right < len(nodes) && nodes[right].cmp(nodes[smallest]) <= 0 {
			smallest = right
		}
		if smallest == i {
			break
		}
		nodes[smallest], nodes[i] = nodes[i], nodes[smallest]
		i = smallest
	}
	return n, true
}
