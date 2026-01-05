package doc

import (
	"fmt"
	"time"

	"github.com/hzerrad/cronic/internal/crontab"
	"github.com/hzerrad/cronic/internal/cronx"
	"github.com/hzerrad/cronic/internal/human"
)

// Generator generates documentation from crontab entries
type Generator struct {
	parser    cronx.Parser
	scheduler cronx.Scheduler
	locale    string
}

// NewGenerator creates a new documentation generator
func NewGenerator(locale string) *Generator {
	return &Generator{
		parser:    cronx.NewParserWithLocale(locale),
		scheduler: cronx.NewScheduler(),
		locale:    locale,
	}
}

// Document represents a complete documentation structure
type Document struct {
	Title       string
	GeneratedAt time.Time
	Source      string
	Jobs        []JobDocument
	Metadata    Metadata
}

// JobDocument represents documentation for a single job
type JobDocument struct {
	LineNumber  int
	Expression  string
	Description string
	Command     string
	Comment     string
	NextRuns    []time.Time
	Warnings    []string
	Stats       *JobStats
}

// JobStats contains frequency statistics for a job
type JobStats struct {
	RunsPerDay  int
	RunsPerHour int
}

// Metadata contains additional document metadata
type Metadata struct {
	TotalJobs   int
	ValidJobs   int
	InvalidJobs int
}

// GenerateDocument generates documentation from crontab entries
func (g *Generator) GenerateDocument(entries []*crontab.Entry, source string, options GenerateOptions) (*Document, error) {
	doc := &Document{
		Title:       "Crontab Documentation",
		GeneratedAt: time.Now(),
		Source:      source,
		Jobs:        []JobDocument{},
		Metadata: Metadata{
			TotalJobs:   0,
			ValidJobs:   0,
			InvalidJobs: 0,
		},
	}

	// Process each entry
	for _, entry := range entries {
		if entry.Type != crontab.EntryTypeJob || entry.Job == nil {
			continue
		}

		doc.Metadata.TotalJobs++

		jobDoc := JobDocument{
			LineNumber: entry.Job.LineNumber,
			Expression: entry.Job.Expression,
			Command:    entry.Job.Command,
			Comment:    entry.Job.Comment,
		}

		if !entry.Job.Valid {
			doc.Metadata.InvalidJobs++
			jobDoc.Description = fmt.Sprintf("Invalid expression: %s", entry.Job.Error)
			doc.Jobs = append(doc.Jobs, jobDoc)
			continue
		}

		doc.Metadata.ValidJobs++

		// Generate human-readable description
		schedule, err := g.parser.Parse(entry.Job.Expression)
		if err == nil {
			humanizer := human.NewHumanizer()
			jobDoc.Description = humanizer.Humanize(schedule)
		}

		// Get next runs if requested
		if options.IncludeNext > 0 {
			times, err := g.scheduler.Next(entry.Job.Expression, time.Now(), options.IncludeNext)
			if err == nil {
				jobDoc.NextRuns = times
			}
		}

		// Get warnings if requested
		if options.IncludeWarnings {
			// This would require validator integration - simplified for now
			jobDoc.Warnings = []string{}
		}

		// Get stats if requested
		if options.IncludeStats {
			stats := g.calculateJobStats(entry.Job.Expression)
			jobDoc.Stats = stats
		}

		doc.Jobs = append(doc.Jobs, jobDoc)
	}

	return doc, nil
}

// calculateJobStats calculates frequency statistics for a job
func (g *Generator) calculateJobStats(expression string) *JobStats {
	// Calculate runs per day
	// Start from just before midnight to capture the first run at midnight
	startTime := ReferenceDate
	queryTime := startTime.Add(-1 * time.Second)
	endTime := startTime.Add(24 * time.Hour) // Using literal, OneDay constant is in stats package

	times, err := g.scheduler.Next(expression, queryTime, 2000) // Using literal, constant is in check package
	if err != nil {
		return nil
	}

	runsPerDay := 0
	for _, t := range times {
		if !t.Before(endTime) {
			break
		}
		if !t.Before(startTime) {
			runsPerDay++
		}
	}

	// Calculate runs per hour
	hourEndTime := startTime.Add(1 * time.Hour)
	hourTimes, err := g.scheduler.Next(expression, queryTime, 100) // Using literal, constant is in check package
	runsPerHour := 0
	if err == nil {
		for _, t := range hourTimes {
			if !t.Before(hourEndTime) {
				break
			}
			if !t.Before(startTime) {
				runsPerHour++
			}
		}
	}

	return &JobStats{
		RunsPerDay:  runsPerDay,
		RunsPerHour: runsPerHour,
	}
}

// GenerateOptions contains options for document generation
type GenerateOptions struct {
	IncludeNext     int  // Number of next runs to include (0 = disabled)
	IncludeWarnings bool // Include validation warnings
	IncludeStats    bool // Include frequency statistics
}
