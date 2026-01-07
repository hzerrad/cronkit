package budget

import (
	"fmt"
	"sort"
	"time"

	"github.com/hzerrad/cronkit/internal/crontab"
	"github.com/hzerrad/cronkit/internal/cronx"
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

// BudgetResult represents the analysis result for a single budget
type BudgetResult struct {
	Budget     Budget
	MaxFound   int         // Maximum concurrent jobs found in the time window
	Passed     bool        // Whether the budget passed
	Violations []Violation // All violations found
}

// BudgetReport represents the complete budget analysis report
type BudgetReport struct {
	Budgets    []BudgetResult
	Passed     bool        // Overall status (true if all budgets passed)
	Violations []Violation // All violations across all budgets
}

// AnalyzeBudget analyzes a crontab against budget rules
func AnalyzeBudget(jobs []*crontab.Job, budgets []Budget, scheduler cronx.Scheduler, parser cronx.Parser) (*BudgetReport, error) {
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
		result, err := analyzeSingleBudget(jobs, budget, scheduler, parser)
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

// analyzeSingleBudget analyzes a crontab against a single budget rule
func analyzeSingleBudget(jobs []*crontab.Job, budget Budget, scheduler cronx.Scheduler, parser cronx.Parser) (*BudgetResult, error) {
	result := &BudgetResult{
		Budget:     budget,
		MaxFound:   0,
		Passed:     true,
		Violations: []Violation{},
	}

	// Filter valid jobs
	validJobs := make([]*crontab.Job, 0, len(jobs))
	for _, job := range jobs {
		if job.Valid {
			validJobs = append(validJobs, job)
		}
	}

	if len(validJobs) == 0 {
		// No valid jobs, budget passes
		return result, nil
	}

	// Find maximum concurrent jobs by examining all run times
	// We need to count jobs that run at the same time, not just overlaps
	startTime := time.Now().Truncate(time.Minute)
	endTime := startTime.Add(budget.TimeWindow)

	// Collect all run times for all jobs, grouped by minute
	type jobRun struct {
		time  time.Time
		jobID string
	}
	var allRuns []jobRun

	for _, job := range validJobs {
		jobID := fmt.Sprintf("line-%d", job.LineNumber)
		if job.LineNumber == 0 {
			jobID = job.Expression
		}

		times, err := scheduler.Next(job.Expression, startTime.Add(-1*time.Second), 10000)
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

	// Group runs by time (minute precision) to find max concurrent
	timeMap := make(map[time.Time]map[string]bool)
	for _, run := range allRuns {
		if timeMap[run.time] == nil {
			timeMap[run.time] = make(map[string]bool)
		}
		timeMap[run.time][run.jobID] = true
	}

	// Find maximum concurrent jobs
	result.MaxFound = 0
	for t, jobs := range timeMap {
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
			violation := Violation{
				Time:   t,
				Count:  count,
				Jobs:   jobList,
				Budget: budget,
			}
			result.Violations = append(result.Violations, violation)
		}
	}

	// If no runs found, set to 0
	if result.MaxFound == 0 && len(validJobs) > 0 {
		// Jobs exist but no runs in the time window - conservative estimate
		result.MaxFound = len(validJobs)
	}

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
