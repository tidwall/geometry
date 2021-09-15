// Copyright 2021 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package geometry

import (
	"encoding/binary"
	"math"
	"sort"
	"sync"
)

const qMaxItems = 12
const qMaxDepth = 64

type qNode struct {
	split bool
	items []int
	quads [4]*qNode
}

func (n *qNode) insert(series *baseSeries, bounds, rect Rect, item, depth int) {
	if depth == qMaxDepth {
		// limit depth and insert now
		n.items = append(n.items, item)
	} else if n.split {
		// qnode is split so try to insert into a quad
		q := chooseQuad(bounds, rect)
		if q == -1 {
			// insert into overflow
			n.items = append(n.items, item)
		} else {
			// insert into quad
			qbounds := quadBounds(bounds, q)
			if n.quads[q] == nil {
				n.quads[q] = new(qNode)
			}
			n.quads[q].insert(series, qbounds, rect, item, depth+1)
		}
	} else if len(n.items) == qMaxItems {
		// split qnode, keep current items in place
		var nitems []int
		for i := 0; i < len(n.items); i++ {
			iitem := n.items[i]
			irect := series.SegmentAt(int(iitem)).Rect()
			q := chooseQuad(bounds, irect)
			if q == -1 {
				nitems = append(nitems, iitem)
			} else {
				qbounds := quadBounds(bounds, q)
				if n.quads[q] == nil {
					n.quads[q] = new(qNode)
				}
				n.quads[q].insert(series, qbounds, irect, int(iitem), depth+1)
			}
		}
		n.items = nitems
		n.split = true
		n.insert(series, bounds, rect, item, depth)
	} else {
		n.items = append(n.items, item)
	}
}

func chooseQuad(bounds, rect Rect) int {
	midX := (bounds.Min.X + bounds.Max.X) / 2
	midY := (bounds.Min.Y + bounds.Max.Y) / 2
	if rect.Max.X < midX {
		if rect.Max.Y < midY {
			return 2
		}
		if rect.Min.Y < midY {
			return -1
		}
		return 0
	}
	if rect.Min.X < midX {
		return -1
	}
	if rect.Max.Y < midY {
		return 3
	}
	if rect.Min.Y < midY {
		return -1
	}
	return 1
}

func quadBounds(bounds Rect, q int) Rect {
	centerX := (bounds.Min.X + bounds.Max.X) / 2
	centerY := (bounds.Min.Y + bounds.Max.Y) / 2
	switch q {
	case 0:
		bounds.Min.Y = centerY
		bounds.Max.X = centerX
	case 1:
		bounds.Min.X = centerX
		bounds.Min.Y = centerY
	case 2:
		bounds.Max.X = centerX
		bounds.Max.Y = centerY
	case 3:
		bounds.Min.X = centerX
		bounds.Max.Y = centerY
	}
	return bounds
}

func appendUvarint(dst []byte, x uint64) []byte {
	if x < 0x80 {
		return append(dst, byte(x))
	}
	var data [10]byte
	n := binary.PutUvarint(data[:], x)
	return append(dst, data[:n]...)
}

// fast alternative version of binary.Uvarint. this one gets inlined.
func readUvarint(data []byte, addr int) (uint64, int) {
	item := uint64(data[addr])
	addr++
	if item < 0x80 {
		return item, addr
	}
	item &= 0x7f
	s := 7
loop:
	b := uint64(data[addr])
	addr++
	if b < 0x80 {
		return item | b<<s, addr
	}
	item |= (b & 0x7f) << s
	s += 7
	if s == 70 {
		return 0, -1
	}
	goto loop
}

// compress the quadtree node-pointer-tree into a single bytes array
func (n *qNode) compress(dst []byte) []byte {
	sort.Ints(n.items)
	dst = appendUvarint(dst, uint64(len(n.items)))
	var last int
	for i := 0; i < len(n.items); i++ {
		item := n.items[i]
		dst = appendUvarint(dst, uint64(item-last))
		last = item
	}
	if !n.split {
		// no-quads
		dst = append(dst, 0)
	} else {
		// yes-quads
		dst = append(dst, 1)
		for q := 0; q < 4; q++ {
			if n.quads[q] != nil {
				dst2 := n.quads[q].compress(nil)
				dst = appendUvarint(dst, uint64(len(dst2)))
				dst = append(dst, dst2...)
			} else {
				dst = append(dst, 0)
			}
		}
	}
	return dst
}

// qCompressSearch performs a search on the compressed quadtree.
func qCompressSearch(
	data []byte,
	addr int,
	series *baseSeries,
	bounds Rect,
	rect Rect,
	iter func(seg Segment, item int) bool,
) bool {
	var nitems uint64
	nitems, addr = readUvarint(data, addr)
	var last uint64
	for i := uint64(0); i < nitems; i++ {
		var item uint64
		item, addr = readUvarint(data, addr)
		item += last
		seg := series.SegmentAt(int(item))
		srect := seg.Rect()
		if srect.IntersectsRect(rect) {
			if !iter(seg, int(item)) {
				return false
			}
		}
		last = item
	}
	if data[addr] == 1 {
		addr++
		for q := 0; q < 4; q++ {
			var item uint64
			item, addr = readUvarint(data, addr)
			if item == 0 {
				// empty quad
				continue
			}
			qsize := item
			qbounds := quadBounds(bounds, q)
			if qbounds.IntersectsRect(rect) {
				if !qCompressSearch(data, addr, series, qbounds, rect, iter) {
					return false
				}
			}
			addr += int(qsize)
		}
	}
	return true
}

var qpool = sync.Pool{
	New: func() interface{} {
		q := queue(make([]qnode, 0, 64))
		return &q
	},
}

func qCompressNearbySegment(
	data []byte, addr int, series *baseSeries, bounds Rect,
	distToRect func(rect Rect) float64,
	distToSegment func(seg Segment) float64,
) (Segment, int, float64) {
	q := qpool.Get().(*queue)
	*q = (*q)[:0]
	defer func() { qpool.Put(q) }()
outer_loop:
	for {
		var nearSeg qnode
		var nearSet bool
		var nitems uint64
		nitems, addr = readUvarint(data, addr)
		var last uint64
		for i := uint64(0); i < nitems; i++ {
			var item uint64
			item, addr = readUvarint(data, addr)
			item += last
			seg := series.SegmentAt(int(item))
			dist := distToSegment(seg)
			if !nearSet || dist < nearSeg.dist {
				nearSeg = qnode{
					kind: qseg,
					dist: dist,
					a:    seg.A,
					b:    seg.B,
					pos:  int(item),
				}
				nearSet = true
			}
			last = item
		}
		if nearSet {
			q.push(nearSeg)
		}
		split := data[addr] == 1
		addr++
		if split {
			for i := 0; i < 4; i++ {
				var item uint64
				item, addr = readUvarint(data, addr)
				if item == 0 {
					// empty quad
					continue
				}
				qsize := item
				qbounds := quadBounds(bounds, i)
				dist := distToRect(qbounds)
				nearRect := qnode{
					kind: qrect,
					dist: dist,
					a:    qbounds.Min,
					b:    qbounds.Max,
					pos:  int(addr),
				}
				q.push(nearRect)
				addr += int(qsize)
			}
		}
		for {
			node, ok := q.pop()
			if !ok {
				return Segment{}, -1, math.NaN()
			}
			switch node.kind {
			case qseg:
				return Segment{A: node.a, B: node.b}, node.pos, node.dist
			case qrect:
				addr = node.pos
				bounds = Rect{Min: node.a, Max: node.b}
				continue outer_loop
			}
		}
	}
}
