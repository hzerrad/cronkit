package budget

import (
	"fmt"
	"sort"
	"time"

	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/hzerrad/cronkit/internal/timeutil"
)

// Budget represents a concurrency budget rule
type Budget struct {
	MaxConcurrent int           // Maximum concurrent jobs allowed
	TimeWindow    time.Duration // Time window for budget (e.g., 1m, 1h, 24h)
	Name          string        // Budget name/identifier (optional)
}

// Violation represents a budget violation at a specific time
type Violation struct {
	Time   time.Time
	Count  int      // Number of concurrent jobs
	Jobs   []string // Job identifiers involved
	Budget Budget   // The budget that was violated
}

// UnresolvedItem names a schedule whose Timezone didn't resolve; counted toward MaxFound, not excluded.
type UnresolvedItem struct {
	Expression string
	Locator    inventory.Locator
	Reason     string
}

// BudgetResult represents the analysis result for a single budget
type BudgetResult struct {
	Budget     Budget
	MaxFound   int         // Maximum concurrent jobs found in the time window
	Passed     bool        // Whether the budget passed
	Violations []Violation // All violations found
	// Unresolved lists items that widened MaxFound conservatively instead of a precise per-minute count.
	Unresolved []UnresolvedItem
}

// BudgetReport represents the complete budget analysis report
type BudgetReport struct {
	Budgets    []BudgetResult
	Passed     bool        // Overall status (true if all budgets passed)
	Violations []Violation // All violations across all budgets
}

// AnalyzeBudget analyzes active inventory items against budget rules starting
// at from; callers must run items through inventory.ResolveTimezones first.
func AnalyzeBudget(items []inventory.Item, from time.Time, budgets []Budget, scheduler cronx.Scheduler, parser cronx.Parser) (*BudgetReport, error) {
	if len(budgets) == 0 {
		return nil, fmt.Errorf("no budgets specified")
	}

	report := &BudgetReport{
		Budgets:    []BudgetResult{},
		Passed:     true,
		Violations: []Violation{},
	}

	// Analyze each budget
	for _, budget := range budgets {
		result, err := analyzeSingleBudget(items, from, budget, scheduler, parser)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze budget %s: %w", budget.Name, err)
		}

		report.Budgets = append(report.Budgets, *result)

		// If any budget failed, overall status is failed
		if !result.Passed {
			report.Passed = false
		}

		// Collect all violations
		report.Violations = append(report.Violations, result.Violations...)
	}

	return report, nil
}

// analyzeSingleBudget analyzes items against a single budget rule, comparing
// each item's occurrences (evaluated in its own timezone) in absolute time.
func analyzeSingleBudget(items []inventory.Item, from time.Time, budget Budget, scheduler cronx.Scheduler, parser cronx.Parser) (*BudgetResult, error) {
	result := &BudgetResult{
		Budget:     budget,
		MaxFound:   0,
		Passed:     true,
		Violations: []Violation{},
		Unresolved: []UnresolvedItem{},
	}

	// An unresolvable-timezone item is a real job whose "when" is unknown, so it widens the estimate.
	for _, item := range items {
		if inventory.IsUnresolvableTimezone(item) {
			result.Unresolved = append(result.Unresolved, UnresolvedItem{
				Expression: item.Expression,
				Locator:    item.Locator,
				Reason:     item.Reason,
			})
		}
	}
	unresolvableTimezones := len(result.Unresolved)

	// @reboot fires at boot, not at a moment this analysis can place in the window, so it's excluded.
	validItems := make([]inventory.Item, 0, len(items))
	for _, item := range items {
		if item.State != inventory.StateActive {
			continue
		}
		if parsed, err := parser.Parse(item.Expression); err == nil && parsed.Kind == cronx.KindReboot {
			continue
		}
		validItems = append(validItems, item)
	}

	if len(validItems) == 0 && unresolvableTimezones == 0 {
		// No active, analysable items, budget passes
		return result, nil
	}

	// Find maximum concurrent items by examining all run times
	// We need to count items that run at the same time, not just overlaps
	startTime := from.Truncate(time.Minute)
	endTime := startTime.Add(budget.TimeWindow)

	// Collect all run times for all items, grouped by minute
	type jobRun struct {
		time  time.Time
		jobID string
	}
	var allRuns []jobRun

	for i, item := range validItems {
		jobID := item.Locator.Identity(i, "job-", "line-")

		loc, err := timeutil.ResolveLocation(item.Timezone, startTime.Location())
		if err != nil {
			continue
		}

		times, err := scheduler.Next(item.Expression, startTime.In(loc), 10000)
		if err != nil {
			continue
		}

		for _, t := range times {
			if t.After(endTime) || t.Equal(endTime) {
				break
			}
			if !t.Before(startTime) {
				allRuns = append(allRuns, jobRun{
					time:  t.Truncate(time.Minute),
					jobID: jobID,
				})
			}
		}
	}

	// Group runs by minute, keyed by instant so runs in different zones collide
	timeMap := make(map[int64]map[string]bool)
	repTime := make(map[int64]time.Time)
	for _, run := range allRuns {
		k := timeutil.MinuteKey(run.time)
		if timeMap[k] == nil {
			timeMap[k] = make(map[string]bool)
		}
		timeMap[k][run.jobID] = true
		if _, seen := repTime[k]; !seen {
			repTime[k] = run.time
		}
	}

	// Find maximum concurrent jobs
	result.MaxFound = 0
	for k, jobs := range timeMap {
		count := len(jobs)
		if count > result.MaxFound {
			result.MaxFound = count
		}

		// Collect violations
		if count > budget.MaxConcurrent {
			jobList := make([]string, 0, len(jobs))
			for jobID := range jobs {
				jobList = append(jobList, jobID)
			}
			sort.Strings(jobList)
			violation := Violation{
				Time:   repTime[k],
				Count:  count,
				Jobs:   jobList,
				Budget: budget,
			}
			result.Violations = append(result.Violations, violation)
		}
	}

	// If no runs found, set to 0
	if result.MaxFound == 0 && len(validItems) > 0 {
		// Items exist but no runs in the time window - conservative estimate
		result.MaxFound = len(validItems)
	}

	// Items with an unresolvable timezone can't be placed in per-minute counts, so they widen the estimate.
	result.MaxFound += unresolvableTimezones

	// Check if budget is violated
	if result.MaxFound > budget.MaxConcurrent {
		result.Passed = false
	} else {
		// Budget passed - clear any violations we might have collected
		result.Violations = []Violation{}
	}

	// Sort violations by time
	sort.Slice(result.Violations, func(i, j int) bool {
		return result.Violations[i].Time.Before(result.Violations[j].Time)
	})

	return result, nil
}
