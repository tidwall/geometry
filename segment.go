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
	segmentSize := seg.A.Distance(seg.B)
	if segmentSize == 0 {
		return seg.A.Distance(point)
	}
	dy := seg.B.Y - seg.A.Y
	dx := seg.B.X - seg.A.X
	return math.Abs(dy*point.X-dx*point.Y+seg.B.X*seg.A.Y-seg.B.Y*seg.A.X) / segmentSize
}
