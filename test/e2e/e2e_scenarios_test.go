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

			It("should be able to use the example command", func() {
				command := exec.Command(pathToCLI, "example")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Hello from cronic!"))
			})
		})

		Context("when a user tries different commands in sequence", func() {
			It("should handle multiple command executions", func() {
				By("checking version first")
				command := exec.Command(pathToCLI, "version")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				By("running example command")
				command = exec.Command(pathToCLI, "example", "--name", "E2E Test")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Hello, E2E Test!"))

				By("checking help for a specific command")
				command = exec.Command(pathToCLI, "help", "example")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
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
				command := exec.Command(pathToCLI, "example", "--invalid-flag")
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
