package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var pathToCLI string

var _ = BeforeSuite(func() {
	var err error
	// Build the CLI binary for testing
	pathToCLI, err = gexec.Build("github.com/hzerrad/cronic/cmd/cronic")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	// Clean up the built binary
	gexec.CleanupBuildArtifacts()
})

var _ = Describe("E2E Scenarios", func() {
	var tempDir string

	BeforeEach(func() {
		var err error
		// Create a temporary directory for each test
		tempDir, err = os.MkdirTemp("", "cronic-e2e-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// Clean up the temporary directory
		if tempDir != "" {
			_ = os.RemoveAll(tempDir) // nolint:errcheck // Test cleanup, ignore errors
		}
	})

	Describe("Complete User Workflow", func() {
		Context("when a new user runs the CLI for the first time", func() {
			It("should display help when no command is provided", func() {
				command := exec.Command(pathToCLI)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Usage:"))
				Expect(session.Out).To(gbytes.Say("Available Commands:"))
			})

			It("should be able to check version", func() {
				command := exec.Command(pathToCLI, "version")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("cronic"))
			})
		})

		Context("when a user tries different commands in sequence", func() {
			It("should handle multiple command executions", func() {
				By("checking version first")
				command := exec.Command(pathToCLI, "version")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("running explain command")
				command = exec.Command(pathToCLI, "explain", "@daily")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("midnight"))

				By("checking help for explain command")
				command = exec.Command(pathToCLI, "help", "explain")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
			})
		})

		Context("when a user plans and validates cron schedules", func() {
			It("should support complete workflow with explain, next, and check", func() {
				By("understanding what a cron expression means")
				command := exec.Command(pathToCLI, "explain", "0 9 * * 1-5")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("09:00"))

				By("checking when the job will actually run")
				command = exec.Command(pathToCLI, "next", "0 9 * * 1-5", "-c", "5")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Next 5 runs"))

				By("validating the expression is correct")
				command = exec.Command(pathToCLI, "check", "0 9 * * 1-5")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("All valid"))

				By("getting machine-readable output for automation")
				command = exec.Command(pathToCLI, "next", "0 9 * * 1-5", "--json", "-c", "3")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say(`"expression"`))
			})

			It("should help DevOps engineer plan backup schedules", func() {
				By("explaining backup schedule frequency")
				command := exec.Command(pathToCLI, "explain", "0 2 * * *")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("verifying next 7 backup times")
				command = exec.Command(pathToCLI, "next", "0 2 * * *", "-c", "7")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("02:00"))
			})

			It("should help developer debug cron timing issues", func() {
				By("checking when hourly job runs")
				command := exec.Command(pathToCLI, "next", "@hourly", "-c", "3")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("comparing with custom interval")
				command = exec.Command(pathToCLI, "next", "0 * * * *", "-c", "3")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
			})

			It("should help DevOps engineer validate crontab before deployment", func() {
				By("validating a crontab file")
				testFile := filepath.Join("..", "..", "testdata", "crontab", "sample.cron")
				command := exec.Command(pathToCLI, "check", "--file", testFile)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("checking invalid expressions are caught")
				command = exec.Command(pathToCLI, "check", "60 0 * * *")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out).To(gbytes.Say("issue"))

				By("getting JSON output for CI/CD integration")
				command = exec.Command(pathToCLI, "check", "0 0 * * *", "--json")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say(`"valid"`))
			})

			It("should help identify DOM/DOW conflicts in schedules", func() {
				By("checking expression with DOM/DOW conflict")
				command := exec.Command(pathToCLI, "check", "0 0 1 * 1", "--verbose")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(2))
				Expect(session.Out).To(gbytes.Say("warning"))

				By("verifying it's valid without verbose flag")
				command = exec.Command(pathToCLI, "check", "0 0 1 * 1")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("All valid"))
			})
		})
	})

	Describe("Error Handling", func() {
		Context("when invalid commands are used", func() {
			It("should provide helpful error messages", func() {
				command := exec.Command(pathToCLI, "nonexistent")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("unknown command"))
			})
		})

		Context("when invalid flags are used", func() {
			It("should provide helpful error messages", func() {
				command := exec.Command(pathToCLI, "explain", "--invalid-flag")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("unknown flag"))
			})
		})
	})

	Describe("File System Interactions", func() {
		Context("when working with files in a temp directory", func() {
			It("should be able to execute commands in different directories", func() {
				// Create a test file in temp directory
				testFile := filepath.Join(tempDir, "test.txt")
				err := os.WriteFile(testFile, []byte("test content"), 0644)
				Expect(err).NotTo(HaveOccurred())

				// Run command in the temp directory
				command := exec.Command(pathToCLI, "version")
				command.Dir = tempDir
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
			})
		})
	})

	Describe("Performance and Reliability", func() {
		Context("when executing commands rapidly", func() {
			It("should handle rapid successive calls", func() {
				for i := 0; i < 5; i++ {
					command := exec.Command(pathToCLI, "version")
					session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(session).Should(gexec.Exit(0))
				}
			})
		})
	})

	Describe("Timeline Visualization Workflows", func() {
		Context("when a DevOps engineer visualizes cron schedules", func() {
			It("should support complete timeline workflow with all features", func() {
				By("creating a timeline for a single expression")
				command := exec.Command(pathToCLI, "timeline", "*/15 * * * *")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Timeline"))

				By("visualizing with custom width")
				command = exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--width", "120")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("visualizing with timezone")
				command = exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--timezone", "UTC")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("visualizing with specific start time")
				command = exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--from", "2025-01-15T00:00:00Z")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("showing overlap information")
				command = exec.Command(pathToCLI, "timeline", "0 * * * *", "--show-overlaps")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("Overlap Summary"))

				By("exporting timeline to file")
				exportFile := filepath.Join(tempDir, "timeline.txt")
				command = exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--export", exportFile)
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				// Verify file was created
				_, err = os.Stat(exportFile)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should help identify job overlaps and conflicts", func() {
				By("creating a timeline from crontab file")
				testFile := filepath.Join("..", "..", "testdata", "crontab", "sample.cron")
				command := exec.Command(pathToCLI, "timeline", "--file", testFile)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("analyzing overlaps in the schedule")
				command = exec.Command(pathToCLI, "timeline", "--file", testFile, "--show-overlaps")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("Overlap Summary"))

				By("getting JSON output for programmatic analysis")
				command = exec.Command(pathToCLI, "timeline", "--file", testFile, "--json")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say(`"overlapStats"`))
			})

			It("should support timezone-aware scheduling analysis", func() {
				By("visualizing schedule in UTC")
				command := exec.Command(pathToCLI, "timeline", "0 9 * * 1-5", "--timezone", "UTC", "--from", "2025-01-15T00:00:00Z")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("comparing with different timezone")
				command = exec.Command(pathToCLI, "timeline", "0 9 * * 1-5", "--timezone", "America/New_York", "--from", "2025-01-15T00:00:00Z")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("exporting timezone-aware timeline")
				exportFile := filepath.Join(tempDir, "timeline-utc.txt")
				command = exec.Command(pathToCLI, "timeline", "0 9 * * 1-5", "--timezone", "UTC", "--export", exportFile)
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				// Verify file was created
				_, err = os.Stat(exportFile)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should support hour view for detailed minute-by-minute analysis", func() {
				By("creating hour view timeline")
				command := exec.Command(pathToCLI, "timeline", "*/5 * * * *", "--view", "hour", "--from", "2025-01-15T14:00:00Z")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("Hour View"))

				By("exporting hour view to JSON")
				exportFile := filepath.Join(tempDir, "timeline-hour.json")
				command = exec.Command(pathToCLI, "timeline", "*/5 * * * *", "--view", "hour", "--json", "--export", exportFile)
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				// Verify file was created
				_, err = os.Stat(exportFile)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when combining timeline with other commands", func() {
			It("should support explain -> timeline -> check workflow", func() {
				By("understanding what the expression means")
				command := exec.Command(pathToCLI, "explain", "*/15 * * * *")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("visualizing the schedule")
				command = exec.Command(pathToCLI, "timeline", "*/15 * * * *")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("validating the expression")
				command = exec.Command(pathToCLI, "check", "*/15 * * * *")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
			})

			It("should support list -> timeline workflow for crontab analysis", func() {
				testFile := filepath.Join("..", "..", "testdata", "crontab", "sample.cron")

				By("listing jobs in crontab")
				command := exec.Command(pathToCLI, "list", "--file", testFile)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("visualizing timeline for the same crontab")
				command = exec.Command(pathToCLI, "timeline", "--file", testFile)
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("analyzing overlaps in the crontab")
				command = exec.Command(pathToCLI, "timeline", "--file", testFile, "--show-overlaps")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
			})
		})

		Context("when exporting timelines for reporting", func() {
			It("should export both text and JSON formats", func() {
				By("exporting text format")
				textFile := filepath.Join(tempDir, "timeline.txt")
				command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--export", textFile)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				// Verify text file
				_, err = os.Stat(textFile)
				Expect(err).NotTo(HaveOccurred())
				content, err := os.ReadFile(textFile)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).To(ContainSubstring("Timeline"))

				By("exporting JSON format")
				jsonFile := filepath.Join(tempDir, "timeline.json")
				command = exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--json", "--export", jsonFile)
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				// Verify JSON file
				_, err = os.Stat(jsonFile)
				Expect(err).NotTo(HaveOccurred())
				content, err = os.ReadFile(jsonFile)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).To(ContainSubstring(`"view"`))
			})

			It("should export with all flags combined", func() {
				exportFile := filepath.Join(tempDir, "timeline-full.txt")
				command := exec.Command(pathToCLI, "timeline", "*/15 * * * *",
					"--width", "120",
					"--timezone", "UTC",
					"--from", "2025-01-15T00:00:00Z",
					"--show-overlaps",
					"--export", exportFile)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				// Verify file was created with all features
				_, err = os.Stat(exportFile)
				Expect(err).NotTo(HaveOccurred())
				content, err := os.ReadFile(exportFile)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).To(ContainSubstring("Timeline"))
				Expect(string(content)).To(ContainSubstring("Overlap Summary"))
			})
		})
	})
})
