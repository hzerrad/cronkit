package human_test

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/human"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	minuteSyntaxByKind = []string{"*", "*/15", "10-20/5", "30", "10-20", "1,31"}
	hourSyntaxByKind   = []string{"*", "*/6", "9-17/2", "9", "9-17", "9,12,17"}

	everyNMinutes = regexp.MustCompile(`^Every (\d+) minutes?$`)
	atExactTime   = regexp.MustCompile(`^At (\d{2}):(\d{2})$`)
)

func allKindPairs() []string {
	var expressions []string
	for _, m := range minuteSyntaxByKind {
		for _, h := range hourSyntaxByKind {
			expressions = append(expressions, fmt.Sprintf("%s %s * * *", m, h))
		}
	}
	return expressions
}

func TestEveryKindPairIsDescribed(t *testing.T) {
	parser := cronx.NewParser()
	humanizer := human.NewHumanizer()

	for _, expr := range allKindPairs() {
		t.Run(expr, func(t *testing.T) {
			schedule, err := parser.Parse(expr)
			require.NoError(t, err)

			description := humanizer.Humanize(schedule)
			assert.NotContains(t, description, "Runs periodically",
				"%q fell through to the fallback", expr)
		})
	}
}

func TestDescriptionMatchesSchedule(t *testing.T) {
	parser := cronx.NewParser()
	humanizer := human.NewHumanizer()
	scheduler := cronx.NewScheduler()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, expr := range allKindPairs() {
		t.Run(expr, func(t *testing.T) {
			schedule, err := parser.Parse(expr)
			require.NoError(t, err)

			description := humanizer.Humanize(schedule)
			runs, err := scheduler.Next(expr, from, 50)
			require.NoError(t, err)

			if m := everyNMinutes.FindStringSubmatch(description); m != nil {
				interval, _ := strconv.Atoi(m[1])
				for i := 1; i < len(runs); i++ {
					gap := runs[i].Sub(runs[i-1])
					assert.Equal(t, time.Duration(interval)*time.Minute, gap,
						"%q claims %q but %s to %s is %s",
						expr, description, runs[i-1], runs[i], gap)
				}
			}

			if m := atExactTime.FindStringSubmatch(description); m != nil {
				hour, _ := strconv.Atoi(m[1])
				minute, _ := strconv.Atoi(m[2])
				for _, run := range runs {
					assert.Equal(t, hour, run.Hour(), "%q claims %q, got run at %s", expr, description, run)
					assert.Equal(t, minute, run.Minute(), "%q claims %q, got run at %s", expr, description, run)
				}
			}
		})
	}
}
