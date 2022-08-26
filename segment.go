// Copyright 2021 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package geometry

import "math"

// Segment is a two point line
type Segment struct {
	A, B Point
}

// Move a segment by delta
func (seg Segment) Move(deltaX, deltaY float64) Segment {
	return Segment{
		A: Point{X: seg.A.X + deltaX, Y: seg.A.Y + deltaY},
		B: Point{X: seg.B.X + deltaX, Y: seg.B.Y + deltaY},
	}
}

// Rect is the outer boundaries of the segment.
func (seg Segment) Rect() Rect {
	var rect Rect
	rect.Min = seg.A
	rect.Max = seg.B
	if rect.Min.X > rect.Max.X {
		rect.Min.X, rect.Max.X = rect.Max.X, rect.Min.X
	}
	if rect.Min.Y > rect.Max.Y {
		rect.Min.Y, rect.Max.Y = rect.Max.Y, rect.Min.Y
	}
	return rect
}

func (seg Segment) CollinearPoint(point Point) bool {
	cmpxr := seg.GetCollinearity(point)
	return cmpxr == 0
}

func (seg Segment) GetCollinearity(point Point) float64 {
	cmpx, cmpy := point.X-seg.A.X, point.Y-seg.A.Y
	rx, ry := seg.B.X-seg.A.X, seg.B.Y-seg.A.Y
	cmpxr := cmpx*ry - cmpy*rx
	return cmpxr
}

func (seg Segment) ContainsPoint(point Point) bool {
	return seg.Raycast(point).On
}

// IntersectsSegment detects if segment intersects with other segment
func (seg Segment) IntersectsSegment(other Segment) bool {
	a, b, c, d := seg.A, seg.B, other.A, other.B
	// do the bounding boxes intersect?
	if a.Y > b.Y {
		if c.Y > d.Y {
			if b.Y > c.Y || a.Y < d.Y {
				return false
			}
		} else {
			if b.Y > d.Y || a.Y < c.Y {
				return false
			}
		}
	} else {
		if c.Y > d.Y {
			if a.Y > c.Y || b.Y < d.Y {
				return false
			}
		} else {
			if a.Y > d.Y || b.Y < c.Y {
				return false
			}
		}
	}
	if a.X > b.X {
		if c.X > d.X {
			if b.X > c.X || a.X < d.X {
				return false
			}
		} else {
			if b.X > d.X || a.X < c.X {
				return false
			}
		}
	} else {
		if c.X > d.X {
			if a.X > c.X || b.X < d.X {
				return false
			}
		} else {
			if a.X > d.X || b.X < c.X {
				return false
			}
		}
	}
	if seg.A == other.A || seg.A == other.B ||
		seg.B == other.A || seg.B == other.B {
		return true
	}

	// the following code is from http://ideone.com/PnPJgb
	cmpx, cmpy := c.X-a.X, c.Y-a.Y
	rx, ry := b.X-a.X, b.Y-a.Y
	cmpxr := cmpx*ry - cmpy*rx
	if cmpxr == 0 {
		// Lines are collinear, and so intersect if they have any overlap
		if !(((c.X-a.X <= 0) != (c.X-b.X <= 0)) ||
			((c.Y-a.Y <= 0) != (c.Y-b.Y <= 0))) {
			return seg.Raycast(other.A).On || seg.Raycast(other.B).On
			//return false
		}
		return true
	}
	sx, sy := d.X-c.X, d.Y-c.Y
	cmpxs := cmpx*sy - cmpy*sx
	rxs := rx*sy - ry*sx
	if rxs == 0 {
		return false // segments are parallel.
	}
	rxsr := 1 / rxs
	t := cmpxs * rxsr
	u := cmpxr * rxsr
	return (t >= 0) && (t <= 1) && (u >= 0) && (u <= 1)
}

// ContainsSegment returns true if segment contains other segment
func (seg Segment) ContainsSegment(other Segment) bool {
	return seg.Raycast(other.A).On && seg.Raycast(other.B).On
}

// distance from point to line
func (seg Segment) Distance(point Point) float64 {
	segmentSize := seg.GetSize()
	if segmentSize == 0 {
		return seg.A.Distance(point)
	}
	// https://ru.wikipedia.org/wiki/%D0%A0%D0%B0%D1%81%D1%81%D1%82%D0%BE%D1%8F%D0%BD%D0%B8%D0%B5_%D0%BE%D1%82_%D1%82%D0%BE%D1%87%D0%BA%D0%B8_%D0%B4%D0%BE_%D0%BF%D1%80%D1%8F%D0%BC%D0%BE%D0%B9_%D0%BD%D0%B0_%D0%BF%D0%BB%D0%BE%D1%81%D0%BA%D0%BE%D1%81%D1%82%D0%B8
	dy := seg.B.Y - seg.A.Y
	dx := seg.B.X - seg.A.X
	return math.Abs(dy*point.X-dx*point.Y+seg.B.X*seg.A.Y-seg.B.Y*seg.A.X) / segmentSize
}

func (seg Segment) GetSize() float64 {
	return seg.A.Distance(seg.B)
}

// returns nearest point at Line at seg to C
func (seg Segment) GetNearestToPoint(C Point) *Point {
	// seg: ax + by + c = 0
	a := seg.A.Y - seg.B.Y
	if a == 0 {
		return &Point{C.X, seg.A.Y}
	}
	if seg.Distance(C) == 0 {
		return &Point{
			X: C.X,
			Y: C.Y,
		}
	}

	// https://math.semestr.ru/line/perpendicular.php
	// Прямая, проходящая через точку C(x1; y1) и перпендикулярная прямой Ax+By+C=0,
	// представляется уравнением
	// A(y-y1)-B(x-x1)=0 (2)

	b := seg.B.X - seg.A.X
	c := seg.A.X*seg.B.Y - seg.B.X*seg.A.Y

	var rx, ry float64
	ry = (a*a*C.Y - a*b*C.X - b*c) / (a*a + b*b)
	rx = (-1) * (c + b*ry) / a

	return &Point{
		X: rx,
		Y: ry,
	}
}
