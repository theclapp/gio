// SPDX-License-Identifier: Unlicense OR MIT

package layout

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/io/router"
	"gioui.org/op"
)

func TestListPosition(t *testing.T) {
	_s := func(e ...event.Event) []event.Event { return e }
	r := new(router.Router)
	gtx := Context{
		Ops: new(op.Ops),
		Constraints: Constraints{
			Max: image.Pt(20, 10),
		},
		Queue: r,
	}
	el := func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(10, 10)}
	}
	for _, tc := range []struct {
		label  string
		num    int
		scroll []event.Event
		first  int
		count  int
		offset int
		last   int
	}{
		{label: "no item", last: 20},
		{label: "1 visible 0 hidden", num: 1, count: 1, last: 10},
		{label: "2 visible 0 hidden", num: 2, count: 2},
		{label: "2 visible 1 hidden", num: 3, count: 2},
		{label: "3 visible 0 hidden small scroll", num: 3, count: 3, offset: 5, last: -5,
			scroll: _s(
				pointer.Event{
					Source:   pointer.Mouse,
					Buttons:  pointer.ButtonLeft,
					Type:     pointer.Press,
					Position: f32.Pt(0, 0),
				},
				pointer.Event{
					Source: pointer.Mouse,
					Type:   pointer.Scroll,
					Scroll: f32.Pt(5, 0),
				},
				pointer.Event{
					Source:   pointer.Mouse,
					Buttons:  pointer.ButtonLeft,
					Type:     pointer.Release,
					Position: f32.Pt(5, 0),
				},
			)},
		{label: "3 visible 0 hidden small scroll 2", num: 3, count: 3, offset: 3, last: -7,
			scroll: _s(
				pointer.Event{
					Source:   pointer.Mouse,
					Buttons:  pointer.ButtonLeft,
					Type:     pointer.Press,
					Position: f32.Pt(0, 0),
				},
				pointer.Event{
					Source: pointer.Mouse,
					Type:   pointer.Scroll,
					Scroll: f32.Pt(3, 0),
				},
				pointer.Event{
					Source:   pointer.Mouse,
					Buttons:  pointer.ButtonLeft,
					Type:     pointer.Release,
					Position: f32.Pt(5, 0),
				},
			)},
		{label: "2 visible 1 hidden large scroll", num: 3, count: 2, first: 1,
			scroll: _s(
				pointer.Event{
					Source:   pointer.Mouse,
					Buttons:  pointer.ButtonLeft,
					Type:     pointer.Press,
					Position: f32.Pt(0, 0),
				},
				pointer.Event{
					Source: pointer.Mouse,
					Type:   pointer.Scroll,
					Scroll: f32.Pt(10, 0),
				},
				pointer.Event{
					Source:   pointer.Mouse,
					Buttons:  pointer.ButtonLeft,
					Type:     pointer.Release,
					Position: f32.Pt(15, 0),
				},
			)},
	} {
		t.Run(tc.label, func(t *testing.T) {
			gtx.Ops.Reset()

			var list List
			// Initialize the list.
			list.Layout(gtx, tc.num, el)
			// Generate the scroll events.
			r.Frame(gtx.Ops)
			r.Queue(tc.scroll...)
			// Let the list process the events.
			list.Layout(gtx, tc.num, el)

			pos := list.Position
			if got, want := pos.First, tc.first; got != want {
				t.Errorf("List: invalid first position: got %v; want %v", got, want)
			}
			if got, want := pos.Count, tc.count; got != want {
				t.Errorf("List: invalid number of visible children: got %v; want %v", got, want)
			}
			if got, want := pos.Offset, tc.offset; got != want {
				t.Errorf("List: invalid first visible offset: got %v; want %v", got, want)
			}
			if got, want := pos.OffsetLast, tc.last; got != want {
				t.Errorf("List: invalid last visible offset: got %v; want %v", got, want)
			}
		})
	}
}

func TestScrollFunctions(t *testing.T) {
	gtx := Context{
		Ops: new(op.Ops),
		Constraints: Constraints{
			Max: image.Pt(1000, 1000),
		},
	}

	l := List{Axis: Vertical}
	listLen := 1000
	layoutList := func(gtx Context) Dimensions {
		return l.Layout(gtx, listLen, func(gtx Context, i int) Dimensions {
			var dims Dimensions
			switch i {
			case 24:
				// Item is really tall: 3x the window size
				dims.Size = image.Pt(1000, 3000)
			default:
				dims.Size = image.Pt(1000, 100)
			}
			return dims
		})
	}
	checkFirstLast := func(first, last int) {
		t.Helper()
		check(t, first, l.FirstItem())
		check(t, last, l.LastItem())
	}

	t.Run("ScrollTo", func(t *testing.T) {
		dims := layoutList(gtx)
		check(t, image.Pt(1000, 1000), dims.Size)
		checkFirstLast(0, 9)

		// ScrollTo an item that's already in view
		l.ScrollTo(1)
		layoutList(gtx)
		checkFirstLast(0, 9)

		// ScrollTo an item that's not in view -- in this case, should shift down
		// one item.
		l.ScrollTo(10)
		layoutList(gtx)
		checkFirstLast(1, 10)

		l.ScrollTo(25)
		layoutList(gtx)
		checkFirstLast(24, 25)
	})

	t.Run("ScrollPage", func(t *testing.T) {
		// Set top of list to item 1
		l.Position = Position{First: 1, BeforeEnd: true}

		l.PageNext()
		layoutList(gtx)
		checkFirstLast(11, 20)

		l.PagePrev()
		layoutList(gtx)
		checkFirstLast(1, 10)

		// ScrollPage -1 with item 1 displayed first
		l.ScrollPages(-1)
		layoutList(gtx)
		checkFirstLast(0, 9)
	})

	t.Run("Scroll to large item", func(t *testing.T) {
		// Item 24 is 3x as tall as the window: show its bottom.
		l.Position.First = 0
		l.Position.Offset = 0
		layoutList(gtx)
		l.ScrollTo(24)
		layoutList(gtx)
		checkFirstLast(24, 24)
		check(t, 2000, l.Position.Offset)

		// Go there again to show its top. (Could also just set Position.First = 24
		// & Position.Offset = 0.)
		l.ScrollTo(24)
		layoutList(gtx)
		checkFirstLast(24, 24)
		check(t, 0, l.Position.Offset)

		// Scroll 2 pages from top. Item 24 is very tall, so it takes up the rest
		// of the window.
		l.ScrollTo(0)
		layoutList(gtx)
		l.ScrollPages(2)
		layoutList(gtx)
		checkFirstLast(20, 24)

		// Starting from the end of the list, scroll back to item 24: make sure
		// we're at the beginning of the item.
		l.ScrollTo(1000)
		layoutList(gtx)
		l.ScrollTo(24)
		layoutList(gtx)
		checkFirstLast(24, 24)
		check(t, 0, l.Position.Offset)

		// PagePrev works
		l.ScrollTo(1000)
		layoutList(gtx)
		checkFirstLast(990, 999)
		l.PagePrev()
		layoutList(gtx)
		checkFirstLast(980, 989)
	})

	t.Run("ScrollToEnd", func(t *testing.T) {
		l.ScrollToEnd = true
		l.Position.BeforeEnd = false

		// Draw from the end and go back.
		layoutList(gtx)
		checkFirstLast(990, 999)

		// Add an item: still draws at end.
		listLen++
		layoutList(gtx)
		checkFirstLast(991, 1000)

		// Remove the item: still at end.
		listLen--
		layoutList(gtx)
		checkFirstLast(990, 999)

		// PagePrev from end of list works.
		l.PagePrev()
		layoutList(gtx)
		checkFirstLast(980, 989)
		check(t, true, l.Position.BeforeEnd)
	})

	t.Run("Small list", func(t *testing.T) {
		l.ScrollToEnd = false

		t.Run("len=0", func(t *testing.T) {
			listLen = 0
			layoutList(gtx)
			checkFirstLast(0, 0)

			l.ScrollTo(1)
			layoutList(gtx)
			checkFirstLast(0, 0)
		})
		t.Run("len=1", func(t *testing.T) {
			listLen = 1
			layoutList(gtx)
			checkFirstLast(0, 0)

			l.ScrollTo(2)
			layoutList(gtx)
			checkFirstLast(0, 0)
		})
		t.Run("len=5", func(t *testing.T) {
			listLen = 5
			l.ScrollTo(0)
			layoutList(gtx)
			checkFirstLast(0, 4)

			l.ScrollTo(2)
			layoutList(gtx)
			checkFirstLast(0, 4)

			t.Run("ScrollToEnd", func(t *testing.T) {
				l.ScrollToEnd = true
				l.Position.BeforeEnd = false

				layoutList(gtx)
				checkFirstLast(0, 4)

				l.ScrollTo(2)
				layoutList(gtx)
				checkFirstLast(0, 4)

				l.ScrollTo(10)
				layoutList(gtx)
				checkFirstLast(0, 4)
			})
		})
	})
}

func check(t *testing.T, exp, got interface{}) {
	t.Helper()
	if exp != got {
		t.Errorf("Expected %v, got %v", exp, got)
	}
}
