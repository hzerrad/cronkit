package integration_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hzerrad/cronkit/internal/crontab"
)

var _ = Describe("Crontab Reader", func() {
	var reader crontab.Reader

	BeforeEach(func() {
		reader = crontab.NewReader()
	})

	Describe("Reading crontab files", func() {
		Context("when reading a valid crontab file", func() {
			It("should parse all job entries correctly", func() {
				jobs, err := reader.ReadFile("../../testdata/crontab/valid/sample.cron")

				Expect(err).NotTo(HaveOccurred())
				Expect(jobs).NotTo(BeEmpty())

				// Verify we have multiple jobs
				Expect(len(jobs)).To(BeNumerically(">", 5))

				// All parsed jobs should be valid
				for _, job := range jobs {
					Expect(job).NotTo(BeNil())
					Expect(job.LineNumber).To(BeNumerically(">", 0))
					Expect(job.Expression).NotTo(BeEmpty())
					Expect(job.Command).NotTo(BeEmpty())
				}
			})

			It("should correctly parse job expressions", func() {
				jobs, err := reader.ReadFile("../../testdata/crontab/valid/sample.cron")

				Expect(err).NotTo(HaveOccurred())

				// Find specific jobs and verify their expressions
				var backupJob, diskCheckJob *crontab.Job
				for _, job := range jobs {
					if job.Command == "/usr/local/bin/backup.sh" {
						backupJob = job
					}
					if job.Command == "/usr/local/bin/check-disk.sh" {
						diskCheckJob = job
					}
				}

				Expect(backupJob).NotTo(BeNil(), "Backup job should be found")
				Expect(backupJob.Expression).To(Equal("0 2 * * *"))
				Expect(backupJob.Valid).To(BeTrue())

				Expect(diskCheckJob).NotTo(BeNil(), "Disk check job should be found")
				Expect(diskCheckJob.Expression).To(Equal("*/15 * * * *"))
				Expect(diskCheckJob.Comment).To(Equal("Disk monitoring"))
			})

			It("should handle cron aliases", func() {
				jobs, err := reader.ReadFile("../../testdata/crontab/valid/sample.cron")

				Expect(err).NotTo(HaveOccurred())

				// Find alias jobs
				aliasJobs := make(map[string]string)
				for _, job := range jobs {
					if len(job.Expression) > 0 && job.Expression[0] == '@' {
						aliasJobs[job.Expression] = job.Command
					}
				}

				Expect(aliasJobs).NotTo(BeEmpty())
				Expect(aliasJobs).To(HaveKey("@monthly"))
				Expect(aliasJobs).To(HaveKey("@hourly"))
				Expect(aliasJobs["@monthly"]).To(Equal("/usr/local/bin/cleanup.sh"))
			})

			It("should preserve inline comments", func() {
				jobs, err := reader.ReadFile("../../testdata/crontab/valid/sample.cron")

				Expect(err).NotTo(HaveOccurred())

				// Find job with comment
				var jobWithComment *crontab.Job
				for _, job := range jobs {
					if job.Comment != "" {
						jobWithComment = job
						break
					}
				}

				Expect(jobWithComment).NotTo(BeNil(), "Should find at least one job with a comment")
				Expect(jobWithComment.Comment).NotTo(BeEmpty())
			})
		})

		Context("when reading a crontab with invalid entries", func() {
			It("should parse valid entries and mark invalid ones", func() {
				jobs, err := reader.ReadFile("../../testdata/crontab/invalid/invalid.cron")

				Expect(err).NotTo(HaveOccurred(), "Reading should not fail even with invalid entries")
				Expect(jobs).NotTo(BeEmpty())

				validCount := 0
				invalidCount := 0
				for _, job := range jobs {
					if job.Valid {
						validCount++
					} else {
						invalidCount++
						Expect(job.Error).NotTo(BeEmpty(), "Invalid jobs should have error messages")
					}
				}

				Expect(validCount).To(BeNumerically(">", 0), "Should have at least one valid job")
				Expect(invalidCount).To(BeNumerically(">", 0), "Should have at least one invalid job")
			})

			It("should provide error details for invalid entries", func() {
				jobs, err := reader.ReadFile("../../testdata/crontab/invalid/invalid.cron")

				Expect(err).NotTo(HaveOccurred())

				// Find an invalid job
				var invalidJob *crontab.Job
				for _, job := range jobs {
					if !job.Valid {
						invalidJob = job
						break
					}
				}

				Expect(invalidJob).NotTo(BeNil(), "Should find at least one invalid job")
				Expect(invalidJob.Error).NotTo(BeEmpty())
				Expect(invalidJob.LineNumber).To(BeNumerically(">", 0))
			})
		})

		Context("when reading an empty crontab", func() {
			It("should return an empty job list", func() {
				jobs, err := reader.ReadFile("../../testdata/crontab/valid/empty.cron")

				Expect(err).NotTo(HaveOccurred())
				Expect(jobs).To(BeEmpty())
			})
		})

		Context("when reading a non-existent file", func() {
			It("should return an error", func() {
				jobs, err := reader.ReadFile("../../testdata/crontab/does-not-exist.cron")

				Expect(err).To(HaveOccurred())
				Expect(jobs).To(BeNil())
			})
		})
	})

	Describe("Parsing crontab entries", func() {
		Context("when parsing a file with various entry types", func() {
			It("should identify all entry types correctly", func() {
				entries, err := reader.ParseFile("../../testdata/crontab/valid/sample.cron")

				Expect(err).NotTo(HaveOccurred())
				Expect(entries).NotTo(BeEmpty())

				// Count entry types
				typeCounts := make(map[crontab.EntryType]int)
				for _, entry := range entries {
					typeCounts[entry.Type]++
				}

				Expect(typeCounts[crontab.EntryTypeJob]).To(BeNumerically(">", 0), "Should have job entries")
				Expect(typeCounts[crontab.EntryTypeComment]).To(BeNumerically(">", 0), "Should have comment entries")
				Expect(typeCounts[crontab.EntryTypeEnvVar]).To(BeNumerically(">", 0), "Should have env var entries")
				Expect(typeCounts[crontab.EntryTypeEmpty]).To(BeNumerically(">", 0), "Should have empty line entries")
			})

			It("should preserve line numbers for all entries", func() {
				entries, err := reader.ParseFile("../../testdata/crontab/valid/sample.cron")

				Expect(err).NotTo(HaveOccurred())

				// Verify line numbers are sequential and start at 1
				for i, entry := range entries {
					Expect(entry.LineNumber).To(Equal(i + 1))
				}
			})

			It("should preserve raw content for all entries", func() {
				entries, err := reader.ParseFile("../../testdata/crontab/valid/sample.cron")

				Expect(err).NotTo(HaveOccurred())

				for _, entry := range entries {
					Expect(entry.Raw).NotTo(BeNil())
				}
			})
		})
	})

	Describe("Edge cases", func() {
		Context("when handling complex cron commands", func() {
			It("should preserve command with pipes and redirects", func() {
				jobs, err := reader.ReadFile("../../testdata/crontab/valid/sample.cron")

				Expect(err).NotTo(HaveOccurred())

				// Find the complex command
				var complexJob *crontab.Job
				for _, job := range jobs {
					if job.Expression == "0 3 * * *" {
						complexJob = job
						break
					}
				}

				Expect(complexJob).NotTo(BeNil())
				Expect(complexJob.Command).To(ContainSubstring("cd /var/log"))
				Expect(complexJob.Command).To(ContainSubstring("&&"))
				Expect(complexJob.Command).To(ContainSubstring("find"))
			})

			It("should handle commands with arguments", func() {
				jobs, err := reader.ReadFile("../../testdata/crontab/valid/sample.cron")

				Expect(err).NotTo(HaveOccurred())

				// Find job with arguments
				var jobWithArgs *crontab.Job
				for _, job := range jobs {
					if job.Expression == "30 14 * * *" {
						jobWithArgs = job
						break
					}
				}

				Expect(jobWithArgs).NotTo(BeNil())
				Expect(jobWithArgs.Command).To(ContainSubstring("--config"))
				Expect(jobWithArgs.Command).To(ContainSubstring("--verbose"))
			})
		})

		Context("when handling different time specifications", func() {
			It("should handle time ranges", func() {
				jobs, err := reader.ReadFile("../../testdata/crontab/valid/sample.cron")

				Expect(err).NotTo(HaveOccurred())

				// Find job with time range
				var rangeJob *crontab.Job
				for _, job := range jobs {
					if job.Expression == "0 9-17 * * 1-5" {
						rangeJob = job
						break
					}
				}

				Expect(rangeJob).NotTo(BeNil())
				Expect(rangeJob.Valid).To(BeTrue())
			})
		})
	})
})
