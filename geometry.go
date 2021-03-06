// Copyright 2021 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package geometry

// Geometry is a standard geometry
type Geometry interface {
	Rect() Rect
	Empty() bool
	ContainsPoint(point Point) bool
	IntersectsPoint(point Point) bool
	ContainsRect(rect Rect) bool
	IntersectsRect(rect Rect) bool
	ContainsLine(line *Line) bool
	IntersectsLine(line *Line) bool
	ContainsPoly(poly *Poly) bool
	IntersectsPoly(poly *Poly) bool
}

// require conformance
var _ = []Geometry{Point{}, Rect{}, &Line{}, &Poly{}}
