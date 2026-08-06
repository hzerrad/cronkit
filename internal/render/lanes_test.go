package render

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPlanWidths(t *testing.T) {
	t.Run("should give full layout at 80 cols", func(t *testing.T) {
		b := planWidths(80, 13, 16)
		assert.Equal(t, budget{gutter: 13, plot: 48, expr: 16}, b)
		assert.Equal(t, 80, b.gutter+1+b.plot+1+1+b.expr)
	})
	t.Run("should drop expr column below 80", func(t *testing.T) {
		b := planWidths(79, 13, 16)
		assert.Equal(t, budget{gutter: 13, plot: 64, expr: 0}, b)
	})
	t.Run("should cap gutter at 10 below 60", func(t *testing.T) {
		b := planWidths(44, 13, 16)
		assert.Equal(t, budget{gutter: 10, plot: 32, expr: 0}, b)
	})
	t.Run("should clamp tiny labels up to 8", func(t *testing.T) {
		assert.Equal(t, 8, planWidths(80, 3, 0).gutter)
	})
	t.Run("should cap wide labels at 17", func(t *testing.T) {
		assert.Equal(t, 17, planWidths(120, 40, 0).gutter)
	})
	t.Run("should drop expr when plot would fall under 30", func(t *testing.T) {
		b := planWidths(80, 17, 17)
		assert.Equal(t, budget{gutter: 17, plot: 43, expr: 17}, b)
		for total := 80; total <= 140; total++ {
			b := planWidths(total, 17, 17)
			if b.expr > 0 {
				assert.GreaterOrEqual(t, b.plot, 30)
			}
		}
	})
}

func TestCellPos(t *testing.T) {
	start := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	t.Run("should map start to col 0", func(t *testing.T) {
		assert.Equal(t, 0, cellPos(start, start, day, 48))
	})
	t.Run("should map noon to the middle", func(t *testing.T) {
		assert.Equal(t, 24, cellPos(start.Add(12*time.Hour), start, day, 48))
	})
	t.Run("should clamp the last minute into the final cell", func(t *testing.T) {
		assert.Equal(t, 47, cellPos(start.Add(day-time.Minute), start, day, 48))
	})
}

func TestAxisTicks(t *testing.T) {
	start := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)
	t.Run("should place day ticks every 6h plus end label", func(t *testing.T) {
		ticks := axisTicks(DayView, start, 48)
		assert.Equal(t, []tick{
			{0, "00:00"}, {12, "06:00"}, {24, "12:00"}, {36, "18:00"}, {47, "23:59"},
		}, ticks)
	})
	t.Run("should place hour ticks every 15m from the real start hour", func(t *testing.T) {
		hs := time.Date(2026, 8, 6, 9, 0, 0, 0, time.UTC)
		ticks := axisTicks(HourView, hs, 60)
		assert.Equal(t, []tick{
			{0, "09:00"}, {15, "09:15"}, {30, "09:30"}, {45, "09:45"}, {59, "09:59"},
		}, ticks)
	})
}
