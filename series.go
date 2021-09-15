// Copyright 2021 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package geometry

import (
	"encoding/binary"
	"math"
)

// IndexKind is the kind of index to use in the options.
type IndexKind byte

// IndexKind types
const (
	None     IndexKind = 0
	QuadTree IndexKind = 1
)

func (kind IndexKind) String() string {
	switch kind {
	default:
		return "Unknown"
	case None:
		return "None"
	case QuadTree:
		return "QuadTree"
	}
}

// IndexOptions are segment indexing options
type IndexOptions struct {
	Kind      IndexKind
	MinPoints int
}

var (
	DefaultIndexOptions = &IndexOptions{Kind: QuadTree, MinPoints: 64}
	NoIndexing          = &IndexOptions{Kind: None, MinPoints: 0}
)

// Series is just a series of points with utilities for efficiently accessing
// segments from rectangle queries, making stuff like point-in-polygon lookups
// very quick.
type Series interface {
	Rect() Rect
	Empty() bool
	Convex() bool
	Clockwise() bool
	NumPoints() int
	NumSegments() int
	PointAt(index int) Point
	SegmentAt(index int) Segment
	Index() []byte
	RawPoints() []Point
	Closed() bool
	Search(rect Rect, iter func(seg Segment, index int) bool)
}

func seriesCopyPoints(series Series) []Point {
	points := make([]Point, series.NumPoints())
	for i := 0; i < len(points); i++ {
		points[i] = series.PointAt(i)
	}
	return points
}

// baseSeries is a concrete type containing all that is needed to make a Series.
type baseSeries struct {
	closed    bool      // points create a closed shape
	clockwise bool      // points move clockwise
	convex    bool      // points create a convex shape
	indexKind IndexKind // index kind
	index     []byte    // actual index
	rect      Rect      // minumum bounding rectangle
	points    []Point   // original points
}

var _ Series = &baseSeries{}

// makeSeries returns a processed baseSeries.
func makeSeries(
	points []Point, copyPoints, closed bool, opts *IndexOptions,
) baseSeries {
	if opts == nil {
		opts = DefaultIndexOptions
	}
	var series baseSeries
	series.closed = closed
	if copyPoints {
		series.points = make([]Point, len(points))
		copy(series.points, points)
	} else {
		series.points = points
	}
	series.convex, series.rect, series.clockwise = processPoints(points, closed)
	if opts.MinPoints != 0 && len(points) >= opts.MinPoints {
		series.indexKind = opts.Kind
		series.buildIndex()
	}
	return series
}

func (series *baseSeries) RawPoints() []Point {
	return series.points
}

func (series *baseSeries) Index() []byte {
	return series.index
}

func (series *baseSeries) Clockwise() bool {
	return series.clockwise
}

func (series *baseSeries) Move(deltaX, deltaY float64) Series {
	points := make([]Point, len(series.points))
	for i := 0; i < len(series.points); i++ {
		points[i].X = series.points[i].X + deltaX
		points[i].Y = series.points[i].Y + deltaY
	}
	nseries := makeSeries(points, false, series.closed, nil)
	nseries.indexKind = series.indexKind
	if len(series.Index()) > 0 {
		nseries.buildIndex()
	}
	return &nseries
}

// Empty returns true if the series does not take up space.
func (series *baseSeries) Empty() bool {
	if series == nil {
		return true
	}
	return (series.closed && len(series.points) < 3) || len(series.points) < 2
}

// Rect returns the series rectangle
func (series *baseSeries) Rect() Rect {
	return series.rect
}

// Convex returns true if the points create a convex loop or linestring
func (series *baseSeries) Convex() bool {
	return series.convex
}

// Closed return true if the shape is closed
func (series *baseSeries) Closed() bool {
	return series.closed
}

// NumPoints returns the number of points in the series
func (series *baseSeries) NumPoints() int {
	return len(series.points)
}

// PointAt returns the point at index
func (series *baseSeries) PointAt(index int) Point {
	return series.points[index]
}

// Search for segments that intersect the provided rectangle
func (series *baseSeries) Search(
	rect Rect,
	iter func(seg Segment, idx int) bool,
) {
	if len(series.index) == 0 {
		n := series.NumSegments()
		for i := 0; i < n; i++ {
			seg := series.SegmentAt(i)
			if seg.Rect().IntersectsRect(rect) {
				if !iter(seg, i) {
					return
				}
			}
		}
	} else {
		data := series.index
		n := binary.LittleEndian.Uint32(data[1:])
		data = data[:n:n]
		qCompressSearch(data, 5, series, series.rect, rect, iter)
	}
}

// DistanceToSeries returns an arbritary distance to a Series.
// All the calculations are performed within two functions, that must be
// provided by the caller:
// - distToRect to calculate a distance to a Rectangle.
// - distToSegment to calculate a distance to a Segment.
// Returns NaN if the series is empty.
func DistanceToSeries(
	series Series,
	distToRect func(rect Rect) float64,
	distToSegment func(seg Segment) float64,
) (seg Segment, idx int, dist float64) {
	dist = math.NaN()
	index := series.Index()
	base, ok := series.(*baseSeries)
	if !ok || len(index) == 0 {
		n := series.NumSegments()
		for i := 0; i < n; i++ {
			sseg := series.SegmentAt(i)
			sdist := distToSegment(sseg)
			if i == 0 || sdist < dist {
				seg = sseg
				dist = sdist
			}
		}
	} else {
		data := index
		n := binary.LittleEndian.Uint32(data[1:])
		data = data[:n:n]
		// skip over the first 5 bytes.
		// NOTE: only qtrees. There is no R-tree support.
		seg, idx, dist = qCompressNearbySegment(data, 5, base, base.rect,
			distToRect, distToSegment)
	}
	return seg, idx, dist
}

func (series *baseSeries) NumSegments() int {
	if series.closed {
		if len(series.points) < 3 {
			return 0
		}
		if series.points[len(series.points)-1] == series.points[0] {
			return len(series.points) - 1
		}
		return len(series.points)
	}
	if len(series.points) < 2 {
		return 0
	}
	return len(series.points) - 1
}

func (series *baseSeries) SegmentAt(index int) Segment {
	var seg Segment
	seg.A = series.points[index]
	if index == len(series.points)-1 {
		seg.B = series.points[0]
	} else {
		seg.B = series.points[index+1]
	}
	return seg
}

// processPoints tests if the ring is convex, calculates the outer
// rectangle.
func processPoints(points []Point, closed bool) (
	convex bool, rect Rect, clockwise bool,
) {
	if (closed && len(points) < 3) || len(points) < 2 {
		return
	}
	var concave bool
	var dir int
	var a, b, c Point
	var cwc float64

	for i := 0; i < len(points); i++ {
		// process the rectangle inflation
		if i == 0 {
			rect = Rect{points[i], points[i]}
		} else {
			if points[i].X < rect.Min.X {
				rect.Min.X = points[i].X
			} else if points[i].X > rect.Max.X {
				rect.Max.X = points[i].X
			}
			if points[i].Y < rect.Min.Y {
				rect.Min.Y = points[i].Y
			} else if points[i].Y > rect.Max.Y {
				rect.Max.Y = points[i].Y
			}
		}

		// gather some point positions for concave and clockwise detection
		a = points[i]
		if i == len(points)-1 {
			b = points[0]
			c = points[1]
		} else if i == len(points)-2 {
			b = points[i+1]
			c = points[0]
		} else {
			b = points[i+1]
			c = points[i+2]
		}

		// process the clockwise detection
		cwc += (b.X - a.X) * (b.Y + a.Y)

		// process the convex calculation
		if concave {
			continue
		}

		zCrossProduct := (b.X-a.X)*(c.Y-b.Y) - (b.Y-a.Y)*(c.X-b.X)
		if dir == 0 {
			if zCrossProduct < 0 {
				dir = -1
			} else if zCrossProduct > 0 {
				dir = 1
			}
		} else if zCrossProduct < 0 {
			if dir == 1 {
				concave = true
			}
		} else if zCrossProduct > 0 {
			if dir == -1 {
				concave = true
			}
		}
	}
	return !concave, rect, cwc > 0
}

func (series *baseSeries) clearIndex() {
	series.index = nil
}

func (series *baseSeries) setCompressed(data []byte) {
	binary.LittleEndian.PutUint32(data[1:], uint32(len(data)))
	smaller := make([]byte, len(data))
	copy(smaller, data)
	series.index = smaller
}

func (series *baseSeries) buildIndex() {
	if series.index != nil {
		// already built
		return
	}
	root := new(qNode)
	n := series.NumSegments()
	for i := 0; i < n; i++ {
		seg := series.SegmentAt(i)
		root.insert(series, series.rect, seg.Rect(), i, 0)
	}
	series.setCompressed(
		root.compress([]byte{byte(series.indexKind), 0, 0, 0, 0}),
	)
}
