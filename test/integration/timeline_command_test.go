package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
			Expect(session.Out).To(gbytes.Say(`\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`))
		})

		It("should display day view by default", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say(`00:00 → 23:59`))
		})

		It("should display hour view with --view hour", func() {
			command := exec.Command(pathToCLI, "timeline", "*/5 * * * *", "--view", "hour")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say(`\d{2}:00 → \d{2}:59`))
		})
	})

	Context("when running 'cronkit timeline' with descriptor schedules", func() {
		It("should render @every 1h without panicking", func() {
			command := exec.Command(pathToCLI, "timeline", "@every 1h")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say(`\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`))
		})

		It("should report no runs in this window for @reboot without panicking", func() {
			command := exec.Command(pathToCLI, "timeline", "@reboot")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("no runs in this window"))
		})

		It("should render a crontab file mixing @reboot and @every without panicking", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "descriptors.cron")
			command := exec.Command(pathToCLI, "timeline", "--file", testFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring("@reboot"))
			Expect(output).To(ContainSubstring("@every 5m"))
		})

		It("should include @every in JSON, and must not panic on @reboot", func() {
			// RenderJSON omits @reboot from the jobs array; this only checks that building it does not panic.
			testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "descriptors.cron")
			command := exec.Command(pathToCLI, "timeline", "--file", testFile, "--json")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))

			var result map[string]interface{}
			err = json.Unmarshal(session.Out.Contents(), &result)
			Expect(err).NotTo(HaveOccurred())

			jobs := result["jobs"].([]interface{})
			var expressions []string
			for _, job := range jobs {
				expressions = append(expressions, job.(map[string]interface{})["expression"].(string))
			}
			Expect(expressions).To(ContainElement("@every 5m"))
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
			Expect(session.Out).To(gbytes.Say(`\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`))
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
			// A single expression never collides with itself, so use a crontab with jobs that genuinely overlap.
			testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")
			command := exec.Command(pathToCLI, "timeline", "--file", testFile, "--show-overlaps")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring("overlaps:"))
		})

		It("should not show overlap summary without flag", func() {
			// Same overlapping fixture, but the detail section should stay hidden until --show-overlaps is passed.
			testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")
			command := exec.Command(pathToCLI, "timeline", "--file", testFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			output := string(session.Out.Contents())
			Expect(output).NotTo(ContainSubstring("overlaps:"))
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
			Expect(output).To(ContainSubstring("overlaps:"))
		})
	})

	Context("when running 'cronkit timeline' with --width flag", func() {
		It("should use specified width", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--width", "120")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say(`\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`))
		})

		It("should handle narrow width gracefully", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--width", "50")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say(`\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`))
		})

		It("should enforce minimum width", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--width", "20")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say(`\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`))
		})
	})

	Context("when running 'cronkit timeline' with --timezone flag", func() {
		It("should use UTC timezone", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--timezone", "UTC")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say(`\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`))
		})

		It("should use America/New_York timezone", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--timezone", "America/New_York")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say(`\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`))
		})

		It("should use Europe/London timezone", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--timezone", "Europe/London")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say(`\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`))
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
			Expect(session.Out).To(gbytes.Say(`\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`))
		})

		It("should work with timezone and crontab file", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")
			command := exec.Command(pathToCLI, "timeline", "--file", testFile, "--timezone", "UTC")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say(`\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`))
		})
	})

	Context("when running 'cronkit timeline' with --from flag", func() {
		It("should use specified start time", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--from", "2025-01-15T00:00:00Z",
				"--timezone", "UTC", "--timezone", "UTC")
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
			command := exec.Command(pathToCLI, "timeline", "*/5 * * * *", "--from", "2025-01-15T14:00:00Z", "--view", "hour", "--timezone", "UTC")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			output := string(session.Out.Contents())
			Expect(output).To(MatchRegexp(`14:00 → 14:59`))
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

			_, err = os.Stat(exportFile)
			Expect(err).NotTo(HaveOccurred())

			content, err := os.ReadFile(exportFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(MatchRegexp(`\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`))
		})

		It("should export timeline to JSON file", func() {
			jsonFile := filepath.Join(tempDir, "timeline.json")
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--json", "--export", jsonFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))

			_, err = os.Stat(jsonFile)
			Expect(err).NotTo(HaveOccurred())

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

			Expect(session.Out).To(gbytes.Say(`\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`))

			_, err = os.Stat(exportFile)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should export with show-overlaps flag", func() {
			// A single expression never collides with itself, so export the overlapping sample crontab instead.
			testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")
			command := exec.Command(pathToCLI, "timeline", "--file", testFile, "--export", exportFile, "--show-overlaps")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))

			content, err := os.ReadFile(exportFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("overlaps:"))
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
			Expect(session.Out).To(gbytes.Say(`\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`))
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
				"--timezone", "UTC",
				"--view", "day",
				"--show-overlaps",
				"--export", exportFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))

			_, err = os.Stat(exportFile)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when stdout is not a terminal", func() {
		It("should emit no ANSI escapes by default", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(string(session.Out.Contents())).NotTo(ContainSubstring("\x1b["))
		})

		It("should emit ANSI escapes with --color always", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--color", "always")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(string(session.Out.Contents())).To(ContainSubstring("\x1b["))
		})

		It("should honor COLUMNS for width", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *")
			command.Env = append(os.Environ(), "COLUMNS=60")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			for _, line := range strings.Split(string(session.Out.Contents()), "\n") {
				Expect(len([]rune(line))).To(BeNumerically("<=", 60))
			}
		})
	})

	Context("when running with --ascii", func() {
		It("should stay within 7-bit ASCII", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--ascii")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			for _, b := range session.Out.Contents() {
				Expect(b).To(BeNumerically("<", 128))
			}
		})
	})

	Context("when the window has no runs", func() {
		It("should say so and exit 0", func() {
			// --timezone UTC is pinned so the day-view window isn't anchored to the host's local calendar day.
			command := exec.Command(pathToCLI, "timeline", "0 2 * * 0", "--from", "2026-08-03T00:00:00Z", "--timezone", "UTC")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("no runs in this window"))
		})
	})
})
