// Copyright 2021 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package geometry

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"testing"
	"time"
)

func TestQTreeVarious(t *testing.T) {
	t.Run("max-depth", func(t *testing.T) {
		var n qNode
		for i := 0; i < 100; i++ {
			n.insert(nil, Rect{}, Rect{}, 0, qMaxDepth)
		}
		expect(t, len(n.items) == 100)
	})
}

func TestQTreeSanity(t *testing.T) {
	// Test a bunch of random line strings
	N := 1_000
	points := make([]Point, N+1)
	start := time.Now()
	for time.Since(start) < time.Second/4 {
		func() {
			seed := time.Now().UnixNano()
			// seed = 1624877218367418000
			rng := rand.New(rand.NewSource(seed))
			n := rng.Intn(N)
			if n < 2 {
				n = 2
			}
			points[0].X = rng.Float64()*360 - 180
			points[0].Y = rng.Float64()*180 - 90
			for i := 1; i < n; i++ {
				points[i].X = points[i-1].X + rng.Float64()*1 - 0.5
				points[i].Y = points[i-1].Y + rng.Float64()*1 - 0.5
			}
			ln := NewLine(points[:n], &IndexOptions{Kind: QuadTree, MinPoints: 64})
			if err := qSane(&ln.baseSeries); err != nil {
				t.Fatalf("%s (seed: %d)", err.Error(), seed)
			}
		}()
	}

}

// qSane performs a sanity check on the quadtree.
// The check verifies:
// - All segments exist in the tree.
// - All segments are correctly contained in their respective quads.
func qSane(series *baseSeries) error {
	var addr int
	data := series.Index()
	if len(data) == 0 {
		if series.NumPoints() > 64 {
			return fmt.Errorf("missing index")
		}
		return nil
	} else if series.NumPoints() < 64 {
		return fmt.Errorf("should not have an index")
	}
	if IndexKind(data[0]) != QuadTree {
		return fmt.Errorf("invalid kind byte")
	}
	sz := int(binary.LittleEndian.Uint32(data[1:]))
	if sz != len(data) {
		return fmt.Errorf("invalid size. expected %d, got %d", len(data), sz)
	}
	addr += 5

	idxs, err := qSaneNode(data, addr, series, series.Rect(), 0)
	if err != nil {
		return err
	}
	sort.Ints(idxs)
	nsegs := series.NumSegments()
	if len(idxs) != nsegs {
		return fmt.Errorf("invalid number of segments. expected %d, got %d",
			nsegs, len(idxs))
	}
	for i, j := range idxs {
		if i != j {
			return fmt.Errorf("index mismatch. expected %d, got %d", i, j)
		}
	}
	return nil
}

func qSaneNode(data []byte, addr int, series *baseSeries, bounds Rect, depth int) ([]int, error) {
	if depth > qMaxDepth {
		return nil, fmt.Errorf("max depth reached")
	}
	var idxs []int
	var nitems uint64
	nitems, addr = readUvarint(data, addr)
	var last uint64
	for i := uint64(0); i < nitems; i++ {
		var item uint64
		item, addr = readUvarint(data, addr)
		item += last
		seg := series.SegmentAt(int(item))
		srect := seg.Rect()
		if !bounds.ContainsRect(srect) {
			return nil, fmt.Errorf("segment %d floats outside of its boundary", i)
		}
		idxs = append(idxs, int(item))

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
			if !bounds.ContainsRect(qbounds) {
				return idxs, fmt.Errorf("quad %d floats outside of its boundary", q)
			}
			qidxs, err := qSaneNode(data, addr, series, bounds, depth+1)
			if err != nil {
				return nil, err
			}
			idxs = append(idxs, qidxs...)
			addr += int(qsize)
		}
	}
	return idxs, nil
}

func TestAppendUvarint(t *testing.T) {
	test := func(t *testing.T, a uint64) {
		t.Helper()
		data := appendUvarint(nil, a)
		b, c := readUvarint(data, 0)
		if b != a {
			t.Fatalf("1) for %d, expected %d, got %d", a, a, b)
		}
		if c != len(data) {
			t.Fatalf("2) for %d, expected %d, got %d", a, len(data), c)
		}
	}
	for _, x := range []uint64{
		0x0, 0x1,
		0x7, 0x8, 0xF,
		0xF7, 0xF8, 0xFF,
		0xFF7, 0xFF8, 0xFFF,
		0xFFF7, 0xFFF8, 0xFFFF,
		0xFFFF7, 0xFFFF8, 0xFFFFF,
		0xFFFFF7, 0xFFFFF8, 0xFFFFFF,
		0xFFFFFF7, 0xFFFFFF8, 0xFFFFFFF,
		0xFFFFFFF7, 0xFFFFFFF8, 0xFFFFFFFF,
		0xFFFFFFFF7, 0xFFFFFFFF8, 0xFFFFFFFFF,
		0xFFFFFFFFF7, 0xFFFFFFFFF8, 0xFFFFFFFFFF,
		0xFFFFFFFFFF7, 0xFFFFFFFFFF8, 0xFFFFFFFFFFF,
		0xFFFFFFFFFFF7, 0xFFFFFFFFFFF8, 0xFFFFFFFFFFFF,
		0xFFFFFFFFFFFF7, 0xFFFFFFFFFFFF8, 0xFFFFFFFFFFFFF,
		0xFFFFFFFFFFFFF7, 0xFFFFFFFFFFFFF8, 0xFFFFFFFFFFFFFF,
		0xFFFFFFFFFFFFFF7, 0xFFFFFFFFFFFFFF8, 0xFFFFFFFFFFFFFFF,
		0xFFFFFFFFFFFFFFF7, 0xFFFFFFFFFFFFFFF8, 0xFFFFFFFFFFFFFFFF,
	} {
		test(t, x)
	}

	// test for failure
	data, _ := hex.DecodeString("ffffffffffffffffffff01")
	x, y := readUvarint(data, 0)
	expect(t, x == 0)
	expect(t, y == -1)

}

func TestEmptyIndex(t *testing.T) {
	a, b, c := qCompressNearbySegment([]byte{0, 0}, 0, nil, Rect{}, nil, nil)
	expect(t, a == Segment{})
	expect(t, b == -1)
	expect(t, math.IsNaN(c))
}
