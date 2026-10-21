package core

// Place works in physical screen pixels, including negative monitor origins.
// DPI conversion belongs at the native API boundary, never in this algorithm.
func Place(anchor, work Rect, width, height, gap int) Rect {
	width = min(width, work.Width)
	height = min(height, work.Height)
	x := anchor.X
	y := anchor.Y + anchor.Height + gap
	if y+height > work.Y+work.Height {
		y = anchor.Y - height - gap
	}
	x = max(work.X, min(x, work.X+work.Width-width))
	y = max(work.Y, min(y, work.Y+work.Height-height))
	return Rect{x, y, width, height}
}
