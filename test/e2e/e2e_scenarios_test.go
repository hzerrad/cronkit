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
			_ = os.RemoveAll(tempDir)
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
})
