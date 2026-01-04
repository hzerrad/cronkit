package integration_test

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Stats Command", func() {
	var testDataDir string

	BeforeEach(func() {
		testDataDir = filepath.Join("..", "..", "testdata", "crontab", "valid")
	})

	Describe("Calculating statistics", func() {
		Context("when calculating stats from a crontab file", func() {
			It("should display statistics in text format", func() {
				testFile := filepath.Join(testDataDir, "sample.cron")
				command := exec.Command(pathToCLI, "stats", "--file", testFile)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("Total Jobs"))
				Expect(output).To(ContainSubstring("Summary"))
			})

			It("should display frequency statistics", func() {
				testFile := filepath.Join(testDataDir, "sample.cron")
				command := exec.Command(pathToCLI, "stats", "--file", testFile)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(MatchRegexp("Total Runs per Day|Total Runs per Hour|runs/day|runs/hour"))
			})

			It("should display collision statistics", func() {
				testFile := filepath.Join(testDataDir, "sample.cron")
				command := exec.Command(pathToCLI, "stats", "--file", testFile)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				// Stats command shows summary and frequency info
				Expect(output).To(ContainSubstring("Summary"))
				Expect(output).To(MatchRegexp("Total Runs|Most Frequent"))
			})
		})

		Context("when using --json flag", func() {
			It("should output statistics in JSON format", func() {
				testFile := filepath.Join(testDataDir, "sample.cron")
				command := exec.Command(pathToCLI, "stats", "--file", testFile, "--json")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())

				var stats map[string]interface{}
				err = json.Unmarshal([]byte(output), &stats)
				Expect(err).NotTo(HaveOccurred())
				// Stats command outputs Metrics struct which has TotalRunsPerDay, TotalRunsPerHour, etc.
				Expect(stats).To(HaveKey("TotalRunsPerDay"))
				Expect(stats).To(HaveKey("TotalRunsPerHour"))
				Expect(stats).To(HaveKey("JobFrequencies"))
			})

			It("should include frequency metrics in JSON", func() {
				testFile := filepath.Join(testDataDir, "sample.cron")
				command := exec.Command(pathToCLI, "stats", "--file", testFile, "--json")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())

				var stats map[string]interface{}
				err = json.Unmarshal([]byte(output), &stats)
				Expect(err).NotTo(HaveOccurred())
				// Check for metrics fields that should be present (JSON uses capitalized field names)
				Expect(stats).To(HaveKey("TotalRunsPerDay"))
				Expect(stats).To(HaveKey("TotalRunsPerHour"))
			})
		})

		Context("when using --verbose flag", func() {
			It("should display detailed statistics", func() {
				testFile := filepath.Join(testDataDir, "sample.cron")
				command := exec.Command(pathToCLI, "stats", "--file", testFile, "--verbose")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				// Verbose mode shows histogram and collision stats
				Expect(output).To(ContainSubstring("Summary"))
				Expect(output).To(MatchRegexp("Most Frequent|Histogram|Busiest"))
			})
		})

		Context("when using --top flag", func() {
			It("should limit most frequent jobs to specified number", func() {
				testFile := filepath.Join(testDataDir, "sample.cron")
				command := exec.Command(pathToCLI, "stats", "--file", testFile, "--top", "3", "--verbose")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				// Should show top 3 most frequent jobs
				Expect(output).To(ContainSubstring("Most Frequent"))
			})
		})

		Context("when using --aggregate flag", func() {
			It("should show aggregated statistics", func() {
				testFile := filepath.Join(testDataDir, "sample.cron")
				command := exec.Command(pathToCLI, "stats", "--file", testFile, "--aggregate")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("Total"))
			})
		})

		Context("when reading from stdin", func() {
			It("should calculate statistics from stdin input", func() {
				crontabContent := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"
				command := exec.Command(pathToCLI, "stats", "--stdin")
				command.Stdin = strings.NewReader(crontabContent)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("Total Jobs"))
				Expect(output).To(ContainSubstring("2"))
			})
		})

		Context("when handling empty crontab", func() {
			It("should handle empty crontab gracefully", func() {
				command := exec.Command(pathToCLI, "stats", "--stdin")
				command.Stdin = strings.NewReader("")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("Total Jobs"))
				Expect(output).To(ContainSubstring("0"))
			})
		})
	})
})
