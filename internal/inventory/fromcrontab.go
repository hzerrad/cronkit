package inventory

import "github.com/hzerrad/cronkit/internal/crontab"

// FromCrontabJobs adapts crontab jobs into inventory items, so analyzers can consume them uniformly.
// A crontab job has no timezone of its own (TZ= is a file line, not a Job field), so Timezone stays empty.
func FromCrontabJobs(jobs []*crontab.Job, file string) []Item {
	items := make([]Item, 0, len(jobs))
	for _, job := range jobs {
		item := Item{
			Expression: job.Expression,
			SourceID:   "crontab",
			Dialect:    "vixie",
			Command:    job.Command,
			Shell:      true,
			Comment:    job.Comment,
			State:      StateActive,
			Locator: Locator{
				File: file,
				Line: job.LineNumber,
			},
		}
		if !job.Valid {
			item.State = StateInvalid
			item.Reason = job.Error
		}
		items = append(items, item)
	}
	return items
}
