package human_test

import (
	"strings"
	"testing"
	"time"

	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/human"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var calendarCorpus = []string{
	"0 9 * * *", "0 9 * * 0", "0 9 * * 1", "0 9 * * 1-5", "0 9 * * 1-5/2",
	"0 9 * * */2", "0 9 * * 0-2", "0 9 * * 1,3,5", "0 9 * * 6,0",

	"0 9 1 * *", "0 9 15 * *", "0 9 1-15 * *", "0 9 1-15/5 * *",
	"0 9 */2 * *", "0 9 1,15 * *", "0 9 1-7 * *",

	"0 9 * 1 *", "0 9 * 1-6 *", "0 9 * 1-6/2 *", "0 9 * */2 *",
	"0 9 * 1,6,12 *", "0 9 * 6-8 *",
}

func runSignature(t *testing.T, scheduler cronx.Scheduler, expression string) string {
	t.Helper()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	runs, err := scheduler.Next(expression, from, 60)
	require.NoError(t, err)

	stamps := make([]string, len(runs))
	for i, run := range runs {
		stamps[i] = run.Format(time.RFC3339)
	}
	return strings.Join(stamps, ",")
}

func TestDistinctSchedulesGetDistinctDescriptions(t *testing.T) {
	parser := cronx.NewParser()
	humanizer := human.NewHumanizer()
	scheduler := cronx.NewScheduler()

	type seen struct {
		expression string
		signature  string
	}
	byDescription := map[string]seen{}

	for _, expr := range calendarCorpus {
		schedule, err := parser.Parse(expr)
		require.NoError(t, err)

		description := humanizer.Humanize(schedule)
		signature := runSignature(t, scheduler, expr)

		if prev, ok := byDescription[description]; ok {
			assert.Equal(t, prev.signature, signature,
				"%q and %q both describe as %q but run at different times",
				prev.expression, expr, description)
			continue
		}
		byDescription[description] = seen{expression: expr, signature: signature}
	}
}
