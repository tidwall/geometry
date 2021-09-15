// Copyright 2021 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package geometry

import (
	"math/rand"
	"testing"
)

func TestQueue(t *testing.T) {
	var q queue
	q.push(qnode{
		dist: 2,
		kind: qseg,
	})
	q.push(qnode{
		dist: 1,
	})
	q.push(qnode{
		dist: 5,
	})
	q.push(qnode{
		dist: 3,
	})
	q.push(qnode{
		dist: 4,
	})
	q.push(qnode{
		dist: 2,
		kind: qrect,
	})

	var lastNode qnode
	lastDist := float64(-1)
	var i int
	for {
		node, ok := q.pop()
		if !ok {
			break
		}
		if node.dist < lastDist {
			t.Fatal("queue was out of order")
		}
		if i > 0 {
			if lastNode.cmp(node) > 0 {
				t.Fatal("queue was out of order")
			}
		}
		lastNode = node
		i++
	}
	if i != 6 {
		t.Fatal("queue was wrong size")
	}

	expect(t, qnode{dist: 2, kind: qrect}.cmp(qnode{dist: 2, kind: qseg}) == -1)
	expect(t, qnode{dist: 2, kind: qseg}.cmp(qnode{dist: 2, kind: qrect}) == 1)
	expect(t, qnode{dist: 2, kind: qseg, pos: 1}.cmp(qnode{dist: 2, kind: qseg, pos: 2}) == -1)
	expect(t, qnode{dist: 2, kind: qseg, pos: 2}.cmp(qnode{dist: 2, kind: qseg, pos: 1}) == 1)
	expect(t, qnode{dist: 2, kind: qseg, pos: 2}.cmp(qnode{dist: 2, kind: qseg, pos: 2}) == 0)

}

func BenchmarkQueue(b *testing.B) {
	var q queue

	for i := 0; i < b.N; i++ {
		r := rand.Float64()
		if r < 0.5 {
			q.push(qnode{dist: r})
		} else {
			q.pop()
		}
	}
}
