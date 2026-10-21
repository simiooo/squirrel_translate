//go:build windows

package native

import "testing"

func TestFramelessResizeEdgesAtDifferentScales(t *testing.T) {
	for _, edge := range []int{8, 12, 16} {
		for _, tt := range []struct{ x, y, want int }{{1, 1, 13}, {399, 1, 14}, {1, 299, 16}, {399, 299, 17}, {200, 1, 12}, {200, 299, 15}, {1, 100, 10}, {399, 100, 11}, {200, 100, 0}, {-1, 1, 0}, {400, 100, 0}} {
			if got := frameEdge(tt.x, tt.y, 400, 300, edge); got != tt.want {
				t.Errorf("edge=%d point=%+v got=%d", edge, tt, got)
			}
		}
	}
}
