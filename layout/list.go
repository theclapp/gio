// SPDX-License-Identifier: Unlicense OR MIT

package layout

import (
	"image"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/op"
	"gioui.org/op/clip"
)

type scrollChild struct {
	size image.Point
	call op.CallOp
}

// List displays a subsection of a potentially infinitely
// large underlying list. List accepts user input to scroll
// the subsection.
type List struct {
	Axis Axis
	// ScrollToEnd instructs the list to stay scrolled to the far end position
	// once reached. A List with ScrollToEnd == true and Position.BeforeEnd ==
	// false draws its content with the last item at the end of the list area.
	ScrollToEnd bool
	// If fromEnd is true, then draw fromEndItem at the end of the list area
	// and go back from there, as if ScrollToEnd was true and this item was the
	// last item in the list. If fromEnd is false, fromEndItem is ignored.
	fromEnd     bool
	fromEndItem int

	// Alignment is the cross axis alignment of list elements.
	Alignment Alignment

	cs          Constraints
	scroll      gesture.Scroll
	scrollDelta int

	// Position is updated during Layout. To save the list scroll position,
	// just save Position after Layout finishes. To scroll the list
	// programatically, update Position (e.g. restore it from a saved value)
	// before calling Layout, or use ScrollTo and related functions.
	Position Position

	len int

	// maxSize is the total size of visible children.
	maxSize  int
	children []scrollChild
	dir      iterationDir

	// size is the width or height, in pixels, at the last layout, used in
	// ScrollPages.
	size int
}

// ListElement is a function that computes the dimensions of
// a list element.
type ListElement func(gtx Context, index int) Dimensions

type iterationDir uint8

// Position is a List scroll offset represented as an offset from the top edge
// of a child element.
type Position struct {
	// BeforeEnd tracks whether the List position is before the very end. We
	// use "before end" instead of "at end" so that the zero value of a
	// Position struct is useful.
	//
	// When laying out a list, if ScrollToEnd is true and BeforeEnd is false,
	// then First and Offset are ignored, and the list is drawn with the last
	// item at the bottom. If ScrollToEnd is false then BeforeEnd is ignored.
	BeforeEnd bool
	// First is the index of the first visible child.
	First int
	// last is the index of the last visible child.
	last int
	// Offset is the distance in pixels from the top edge to the child at index
	// First. Positive offsets are before (above or left of) the window edge.
	Offset int
}

const (
	iterateNone iterationDir = iota
	iterateForward
	iterateBackward
)

const inf = 1e6

// init prepares the list for iterating through its children with next.
func (l *List) init(gtx Context, len int) {
	if l.more() {
		panic("unfinished child")
	}
	l.cs = gtx.Constraints
	l.maxSize = 0
	l.children = l.children[:0]
	l.len = len
	l.update(gtx)
	if (!l.fromEnd && l.scrollToEnd()) || l.Position.First > len {
		l.Position.Offset = 0
		l.Position.First = len
	}
}

// Layout the List.
func (l *List) Layout(gtx Context, len int, w ListElement) Dimensions {
	l.init(gtx, len)
	crossMin, crossMax := axisCrossConstraint(l.Axis, gtx.Constraints)
	gtx.Constraints = axisConstraints(l.Axis, 0, inf, crossMin, crossMax)
	macro := op.Record(gtx.Ops)
	for l.next(); l.more(); l.next() {
		child := op.Record(gtx.Ops)
		dims := w(gtx, l.index())
		call := child.Stop()
		l.end(dims, call)
	}
	return l.layout(gtx.Ops, macro)
}

func (l *List) scrollToEnd() bool {
	return l.fromEnd || (l.ScrollToEnd && !l.Position.BeforeEnd)
}

// Dragging reports whether the List is being dragged.
func (l *List) Dragging() bool {
	return l.scroll.State() == gesture.StateDragging
}

// ScrollTo makes sure list index item i is in view.
//
// If it's above the top, it becomes the top item. If it's below the bottom,
// it becomes the bottom item, with said item drawn starting at the end of the
// item. (This means that if the item is taller/longer than the list area, the
// beginning of the item will be out of view.)
//
// If i < 0, uses 0.
//
// If you ScrollTo(n) and then layout a list shorter than n, Layout scrolls to
// the end of the list.
func (l *List) ScrollTo(item int) {
	if item < 0 {
		item = 0
	}

	// Set default.
	l.fromEnd = false

	// If item is already entirely in view, do nothing.
	if l.Position.First < item && item < l.Position.last {
		return
	}

	if (l.Position.First > 0 || l.Position.Offset > 0) && item <= l.Position.First {
		// Item is before, or equal to, the first item drawn. Draw item at
		// offset 0, at the beginning of the list, and go forward.
		l.Position.First = item
		l.Position.Offset = 0
		l.Position.BeforeEnd = true
	} else if item >= l.Position.last {
		// Item is after the last item drawn. Draw the end of item at the end of
		// the list, and go backward.
		l.Position.First = item + 1
		l.fromEnd = true
		l.fromEndItem = item
	}
}

// ScrollPages scrolls a number of pages. n < 0 is up or left, n > 0 is down
// or right. n == 0 is a no-op.
//
// The "page size" is the size of the major axis of the list at its last
// layout. Thus, ScrollPages only works if you've laid out the list at least
// once.
func (l *List) ScrollPages(n int) {
	// If going nowhere, or going backward and we're already at the beginning,
	// or going forward and we're already at the end, do nothing.
	if n == 0 ||
		(n < 0 && l.Position.BeforeEnd && l.Position.First == 0 && l.Position.Offset == 0) ||
		(n > 0 && !l.Position.BeforeEnd) {
		return
	}

	l.Position.Offset += (l.size * n)
	// If you don't do this and l.ScrollToEnd == true, Position.Offset is
	// ignored, so you couldn't ScrollPages(-1) from the end of the list.
	l.Position.BeforeEnd = true
}

func (l *List) PagePrev() {
	l.ScrollPages(-1)
}

func (l *List) PageNext() {
	l.ScrollPages(1)
}

func (l *List) update(gtx Context) {
	d := l.scroll.Scroll(gtx.Metric, gtx, gtx.Now, gesture.Axis(l.Axis))
	l.scrollDelta = d
	l.Position.Offset += d
}

// next advances to the next child.
func (l *List) next() {
	l.dir = l.nextDir()
	// The user scroll offset is applied after scrolling to
	// list end.
	if l.scrollToEnd() && !l.more() && l.scrollDelta < 0 {
		l.Position.BeforeEnd = true
		l.Position.Offset += l.scrollDelta
		l.dir = l.nextDir()
	}
}

// index is current child's position in the underlying list.
func (l *List) index() int {
	switch l.dir {
	case iterateBackward:
		return l.Position.First - 1
	case iterateForward:
		return l.Position.First + len(l.children)
	default:
		panic("Index called before Next")
	}
}

// more reports whether more children are needed.
func (l *List) more() bool {
	return l.dir != iterateNone
}

func (l *List) nextDir() iterationDir {
	_, vsize := axisMainConstraint(l.Axis, l.cs)
	last := l.Position.First + len(l.children)
	// Clamp offset.
	if l.maxSize-l.Position.Offset < vsize &&
		(last == l.len || (l.fromEnd && last == l.fromEndItem+1)) {
		l.Position.Offset = l.maxSize - vsize
	}
	if l.Position.Offset < 0 && l.Position.First == 0 {
		l.Position.Offset = 0
	}
	switch {
	case len(l.children) == l.len:
		return iterateNone
	case l.maxSize-l.Position.Offset < vsize:
		return iterateForward
	case l.Position.Offset < 0:
		return iterateBackward
	}
	return iterateNone
}

// End the current child by specifying its dimensions.
func (l *List) end(dims Dimensions, call op.CallOp) {
	child := scrollChild{dims.Size, call}
	mainSize := axisMain(l.Axis, child.size)
	l.maxSize += mainSize
	switch l.dir {
	case iterateForward:
		l.children = append(l.children, child)
	case iterateBackward:
		l.children = append(l.children, scrollChild{})
		copy(l.children[1:], l.children)
		l.children[0] = child
		l.Position.First--
		l.Position.Offset += mainSize
	default:
		panic("call Next before End")
	}
	l.dir = iterateNone
}

// Layout the List and return its dimensions.
func (l *List) layout(ops *op.Ops, macro op.MacroOp) Dimensions {
	if l.more() {
		panic("unfinished child")
	}
	mainMin, mainMax := axisMainConstraint(l.Axis, l.cs)
	children := l.children
	// Skip invisible children
	for len(children) > 0 {
		sz := children[0].size
		mainSize := axisMain(l.Axis, sz)
		if l.Position.Offset < mainSize {
			break
		}
		l.Position.First++
		l.Position.Offset -= mainSize
		children = children[1:]
	}
	size := -l.Position.Offset
	var maxCross int
	for i, child := range children {
		sz := child.size
		if c := axisCross(l.Axis, sz); c > maxCross {
			maxCross = c
		}
		size += axisMain(l.Axis, sz)
		if size >= mainMax {
			children = children[:i+1]
			break
		}
	}
	pos := -l.Position.Offset
	// ScrollToEnd lists are end aligned.
	if space := mainMax - size; l.ScrollToEnd && space > 0 {
		pos += space
	}
	if len(children) == 0 {
		l.Position.last = 0
	} else {
		l.Position.last = l.Position.First + len(children) - 1
	}
	for _, child := range children {
		sz := child.size
		var cross int
		switch l.Alignment {
		case End:
			cross = maxCross - axisCross(l.Axis, sz)
		case Middle:
			cross = (maxCross - axisCross(l.Axis, sz)) / 2
		}
		childSize := axisMain(l.Axis, sz)
		max := childSize + pos
		if max > mainMax {
			max = mainMax
		}
		min := pos
		if min < 0 {
			min = 0
		}
		r := image.Rectangle{
			Min: axisPoint(l.Axis, min, -inf),
			Max: axisPoint(l.Axis, max, inf),
		}
		stack := op.Push(ops)
		clip.Rect(r).Add(ops)
		op.Offset(FPt(axisPoint(l.Axis, pos, cross))).Add(ops)
		child.call.Add(ops)
		stack.Pop()
		pos += childSize
	}
	atStart := l.Position.First == 0 && l.Position.Offset <= 0
	atEnd := l.Position.First+len(children) == l.len && mainMax >= pos
	if atStart && l.scrollDelta < 0 || atEnd && l.scrollDelta > 0 {
		l.scroll.Stop()
	}
	l.Position.BeforeEnd = !atEnd
	if pos < mainMin {
		pos = mainMin
	}
	if pos > mainMax {
		pos = mainMax
	}
	dims := axisPoint(l.Axis, pos, maxCross)
	call := macro.Stop()
	defer op.Push(ops).Pop()
	pointer.Rect(image.Rectangle{Max: dims}).Add(ops)
	l.scroll.Add(ops)
	call.Add(ops)
	l.fromEnd = false
	l.size = axisMain(l.Axis, dims)
	return Dimensions{Size: dims}
}

// FirstItem returns the index of the first displayed item in the list. In
// the event of large items, FirstItem and LastItem can be the same. FirstItem
// is a convenience function for l.Position.First.
func (l *List) FirstItem() int { return l.Position.First }

// LastItem returns the index of the last displayed item in the list. In the
// event of large items, FirstItem and LastItem can be the same.
func (l *List) LastItem() int { return l.Position.last }
