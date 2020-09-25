package layout

import (
	"image"
	"testing"

	"gioui.org/op"
)

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
	checkFirstLast := func(t *testing.T, first, last int) {
		t.Helper()
		check(t, "first", first, l.Position.First)
		check(t, "last", last, l.Position.last)
	}

	t.Run("ScrollTo", func(t *testing.T) {
		dims := layoutList(gtx)
		check(t, "size", image.Pt(1000, 1000), dims.Size)
		checkFirstLast(t, 0, 9)

		// ScrollTo an item that's already in view
		for i := 0; i < 10; i++ {
			l.ScrollTo(i)
			layoutList(gtx)
			checkFirstLast(t, 0, 9)
		}

		// ScrollTo an item that's not in view -- in this case, should shift down
		// one item.
		l.ScrollTo(10)
		layoutList(gtx)
		checkFirstLast(t, 1, 10)

		// Scroll up to an item still in view -- should shift up one item
		l.ScrollTo(9)
		layoutList(gtx)
		checkFirstLast(t, 1, 10)

		// Scroll down to an item out of view -- should shift down two
		l.ScrollTo(11)
		layoutList(gtx)
		checkFirstLast(t, 2, 11)

		// Scroll deep into the list, then upwards out of the current view -- resulting view
		// should *begin* at index 10, not end there
		l.ScrollTo(20)
		layoutList(gtx)
		checkFirstLast(t, 11, 20)
		l.ScrollTo(10)
		layoutList(gtx)
		checkFirstLast(t, 10, 19)

		l.ScrollTo(25)
		layoutList(gtx)
		checkFirstLast(t, 24, 25)
	})

	t.Run("Scroll to large item", func(t *testing.T) {
		// Item 24 is 3x as tall as the window: show its bottom.
		l.Position.First = 0
		l.Position.Offset = 0
		layoutList(gtx)
		l.ScrollTo(24)
		layoutList(gtx)
		checkFirstLast(t, 24, 24)
		check(t, "offset", 2000, l.Position.Offset)

		// Go there again to show its top. (Could also just set Position.First = 24
		// & Position.Offset = 0.)
		l.ScrollTo(24)
		layoutList(gtx)
		checkFirstLast(t, 24, 24)
		check(t, "offset", 0, l.Position.Offset)

		// Starting from the end of the list, scroll back to item 24: make sure
		// we're at the beginning of the item.
		l.ScrollTo(1000)
		layoutList(gtx)
		l.ScrollTo(24)
		layoutList(gtx)
		checkFirstLast(t, 24, 24)
		check(t, "offset", 0, l.Position.Offset)

	})

	t.Run("ScrollToEnd", func(t *testing.T) {
		l.ScrollToEnd = true
		l.Position.BeforeEnd = false

		// Draw from the end and go back.
		layoutList(gtx)
		checkFirstLast(t, 990, 999)

		// Add an item: still draws at end.
		listLen++
		layoutList(gtx)
		checkFirstLast(t, 991, 1000)

		// Remove the item: still at end.
		listLen--
		layoutList(gtx)
		checkFirstLast(t, 990, 999)
	})

	t.Run("Small list", func(t *testing.T) {
		l.ScrollToEnd = false

		t.Run("len=0", func(t *testing.T) {
			listLen = 0
			layoutList(gtx)
			checkFirstLast(t, 0, 0)

			l.ScrollTo(1)
			layoutList(gtx)
			checkFirstLast(t, 0, 0)
		})
		t.Run("len=1", func(t *testing.T) {
			listLen = 1
			layoutList(gtx)
			checkFirstLast(t, 0, 0)

			l.ScrollTo(2)
			layoutList(gtx)
			checkFirstLast(t, 0, 0)
		})
		t.Run("len=5", func(t *testing.T) {
			listLen = 5
			l.ScrollTo(0)
			layoutList(gtx)
			checkFirstLast(t, 0, 4)

			l.ScrollTo(2)
			layoutList(gtx)
			checkFirstLast(t, 0, 4)

			t.Run("ScrollToEnd", func(t *testing.T) {
				l.ScrollToEnd = true
				l.Position.BeforeEnd = false

				layoutList(gtx)
				checkFirstLast(t, 0, 4)

				l.ScrollTo(2)
				layoutList(gtx)
				checkFirstLast(t, 0, 4)

				l.ScrollTo(10)
				layoutList(gtx)
				checkFirstLast(t, 0, 4)
			})
		})
	})
}

func check(t *testing.T, description string, exp, got interface{}) {
	t.Helper()
	if exp != got {
		t.Errorf("Expected %v, got %v for %s", exp, got, description)
	}
}
