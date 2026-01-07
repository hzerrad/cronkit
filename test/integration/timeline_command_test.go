package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Timeline Command", func() {
	Context("when running 'cronkit timeline' with a valid expression", func() {
		It("should display timeline successfully", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Timeline"))
		})

		It("should display day view by default", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Day View"))
		})

		It("should display hour view with --view hour", func() {
			command := exec.Command(pathToCLI, "timeline", "*/5 * * * *", "--view", "hour")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Hour View"))
		})
	})

	Context("when running 'cronkit timeline' with --json flag", func() {
		It("should output valid JSON", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--json")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))

			var result map[string]interface{}
			err = json.Unmarshal(session.Out.Contents(), &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveKey("view"))
			Expect(result).To(HaveKey("startTime"))
			Expect(result).To(HaveKey("endTime"))
			Expect(result).To(HaveKey("width"))
			Expect(result).To(HaveKey("jobs"))
			Expect(result).To(HaveKey("overlaps"))
		})

		It("should have correct view type in JSON", func() {
			command := exec.Command(pathToCLI, "timeline", "*/5 * * * *", "--view", "hour", "--json")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))

			var result map[string]interface{}
			err = json.Unmarshal(session.Out.Contents(), &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result["view"]).To(Equal("hour"))
		})

		It("should include jobs array in JSON", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--json")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))

			var result map[string]interface{}
			err = json.Unmarshal(session.Out.Contents(), &result)
			Expect(err).NotTo(HaveOccurred())

			jobs := result["jobs"].([]interface{})
			Expect(len(jobs)).To(BeNumerically(">", 0))
		})
	})

	Context("when running 'cronkit timeline' with --file flag", func() {
		It("should read from crontab file", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")
			command := exec.Command(pathToCLI, "timeline", "--file", testFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Timeline"))
		})

		It("should output JSON for crontab file", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")
			command := exec.Command(pathToCLI, "timeline", "--file", testFile, "--json")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))

			var result map[string]interface{}
			err = json.Unmarshal(session.Out.Contents(), &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveKey("jobs"))
		})
	})

	Context("when running 'cronkit timeline' with invalid expression", func() {
		It("should exit with error code 1", func() {
			command := exec.Command(pathToCLI, "timeline", "60 0 * * *")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("invalid"))
		})
	})

	Context("when running 'cronkit timeline' with invalid view", func() {
		It("should exit with error", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--view", "invalid")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("invalid view type"))
		})
	})

	Context("when running 'cronkit timeline' with non-existent file", func() {
		It("should exit with error code 1", func() {
			command := exec.Command(pathToCLI, "timeline", "--file", "/nonexistent/file.cron")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("failed to read"))
		})
	})

	Context("when running 'cronkit timeline' without arguments", func() {
		It("should attempt to read user crontab", func() {
			command := exec.Command(pathToCLI, "timeline")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			// May exit with 0 (empty crontab) or 1 (error reading), but should not crash
			Eventually(session).Should(gexec.Exit())
		})
	})

	Context("when running 'cronkit timeline' with --show-overlaps flag", func() {
		It("should show overlap summary in text output", func() {
			// Create jobs that run at the same time to generate overlaps
			command := exec.Command(pathToCLI, "timeline", "0 * * * *", "--show-overlaps")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring("Overlap Summary"))
		})

		It("should not show overlap summary without flag", func() {
			command := exec.Command(pathToCLI, "timeline", "0 * * * *")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			output := string(session.Out.Contents())
			Expect(output).NotTo(ContainSubstring("Overlap Summary"))
		})

		It("should include overlap statistics in JSON output", func() {
			command := exec.Command(pathToCLI, "timeline", "0 * * * *", "--json")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring(`"overlapStats"`))
			Expect(output).To(ContainSubstring(`"totalWindows"`))
			Expect(output).To(ContainSubstring(`"maxConcurrent"`))
			Expect(output).To(ContainSubstring(`"mostProblematic"`))
		})

		It("should show overlaps with multiple jobs", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")
			command := exec.Command(pathToCLI, "timeline", "--file", testFile, "--show-overlaps")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring("Overlap Summary"))
		})
	})

	Context("when running 'cronkit timeline' with --width flag", func() {
		It("should use specified width", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--width", "120")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Timeline"))
		})

		It("should handle narrow width gracefully", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--width", "50")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Timeline"))
		})

		It("should enforce minimum width", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--width", "20")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			// Should still render successfully with minimum width enforcement
			Expect(session.Out).To(gbytes.Say("Timeline"))
		})
	})

	Context("when running 'cronkit timeline' with --timezone flag", func() {
		It("should use UTC timezone", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--timezone", "UTC")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Timeline"))
		})

		It("should use America/New_York timezone", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--timezone", "America/New_York")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Timeline"))
		})

		It("should use Europe/London timezone", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--timezone", "Europe/London")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Timeline"))
		})

		It("should reject invalid timezone", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--timezone", "Invalid/Timezone")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("invalid timezone"))
		})

		It("should work with timezone and --from flag", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--timezone", "UTC", "--from", "2025-01-15T00:00:00Z")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Timeline"))
		})

		It("should work with timezone and crontab file", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")
			command := exec.Command(pathToCLI, "timeline", "--file", testFile, "--timezone", "UTC")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Timeline"))
		})
	})

	Context("when running 'cronkit timeline' with --from flag", func() {
		It("should use specified start time", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--from", "2025-01-15T00:00:00Z")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring("2025-01-15"))
		})

		It("should reject invalid date format", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--from", "invalid-date")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("invalid --from time format"))
		})

		It("should work with --from and hour view", func() {
			command := exec.Command(pathToCLI, "timeline", "*/5 * * * *", "--from", "2025-01-15T14:00:00Z", "--view", "hour")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring("Hour View"))
			Expect(output).To(ContainSubstring("2025-01-15"))
		})
	})

	Context("when running 'cronkit timeline' with --export flag", func() {
		var tempDir string
		var exportFile string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "cronkit-timeline-test-*")
			Expect(err).NotTo(HaveOccurred())
			exportFile = filepath.Join(tempDir, "timeline.txt")
		})

		AfterEach(func() {
			if tempDir != "" {
				_ = os.RemoveAll(tempDir) // nolint:errcheck // Test cleanup, ignore errors
			}
		})

		It("should export timeline to text file", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--export", exportFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))

			// Check file was created
			_, err = os.Stat(exportFile)
			Expect(err).NotTo(HaveOccurred())

			// Check file content
			content, err := os.ReadFile(exportFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("Timeline"))
		})

		It("should export timeline to JSON file", func() {
			jsonFile := filepath.Join(tempDir, "timeline.json")
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--json", "--export", jsonFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))

			// Check file was created
			_, err = os.Stat(jsonFile)
			Expect(err).NotTo(HaveOccurred())

			// Check file content is valid JSON
			content, err := os.ReadFile(jsonFile)
			Expect(err).NotTo(HaveOccurred())

			var result map[string]interface{}
			err = json.Unmarshal(content, &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveKey("view"))
		})

		It("should export and still print to stdout", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--export", exportFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))

			// Should have output in stdout
			Expect(session.Out).To(gbytes.Say("Timeline"))

			// And file should exist
			_, err = os.Stat(exportFile)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should export with show-overlaps flag", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--export", exportFile, "--show-overlaps")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))

			// Check file content includes overlap summary
			content, err := os.ReadFile(exportFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("Overlap Summary"))
		})

		It("should fail with invalid export path", func() {
			invalidPath := filepath.Join("/nonexistent", "dir", "file.txt")
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--export", invalidPath)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("failed to export"))
		})
	})

	Context("when running 'cronkit timeline' with combined flags", func() {
		It("should work with width, timezone, and from flags", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--width", "100", "--timezone", "UTC", "--from", "2025-01-15T00:00:00Z")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Timeline"))
		})

		It("should work with all flags including export", func() {
			tempDir, err := os.MkdirTemp("", "cronkit-timeline-test-*")
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				_ = os.RemoveAll(tempDir) // nolint:errcheck // Test cleanup, ignore errors
			}()

			exportFile := filepath.Join(tempDir, "timeline.txt")
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *",
				"--width", "120",
				"--timezone", "UTC",
				"--from", "2025-01-15T00:00:00Z",
				"--view", "day",
				"--show-overlaps",
				"--export", exportFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))

			// Check file was created
			_, err = os.Stat(exportFile)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
