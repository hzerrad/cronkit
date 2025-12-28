package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Check Command", func() {
	Context("when running 'cronic check' with a valid expression", func() {
		It("should validate successfully", func() {
			command := exec.Command(pathToCLI, "check", "0 0 * * *")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("All valid"))
		})
	})

	Context("when running 'cronic check' with an invalid expression", func() {
		It("should report errors and exit with code 1", func() {
			command := exec.Command(pathToCLI, "check", "60 0 * * *")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Out).To(gbytes.Say("issue"))
			Expect(session.Out).To(gbytes.Say("ERROR"))
			Expect(session.Out).To(gbytes.Say("CRON-003"))
		})
	})

	Context("when running 'cronic check' with DOM/DOW conflict", func() {
		It("should show as valid without verbose flag", func() {
			command := exec.Command(pathToCLI, "check", "0 0 1 * 1")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("All valid"))
		})

		It("should show warnings with verbose flag", func() {
			command := exec.Command(pathToCLI, "check", "0 0 1 * 1", "--verbose")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(2))
			Expect(session.Out).To(gbytes.Say("warning"))
			Expect(session.Out).To(gbytes.Say("CRON-001"))
			Expect(session.Out).To(gbytes.Say("Hint:"))
		})
	})

	Context("when running 'cronic check --file' with a valid crontab", func() {
		It("should validate successfully", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "sample.cron")
			command := exec.Command(pathToCLI, "check", "--file", testFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
		})
	})

	Context("when running 'cronic check --file' with an invalid crontab", func() {
		It("should report errors and exit with code 1", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "invalid.cron")
			command := exec.Command(pathToCLI, "check", "--file", testFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Out).To(gbytes.Say("issue"))
		})
	})

	Context("when running 'cronic check --file' with non-existent file", func() {
		It("should report error and exit with code 1", func() {
			command := exec.Command(pathToCLI, "check", "--file", "nonexistent.cron")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Out).To(gbytes.Say("Failed to read"))
		})
	})

	Context("when running 'cronic check --json'", func() {
		It("should output valid JSON", func() {
			command := exec.Command(pathToCLI, "check", "0 0 * * *", "--json")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring(`"valid"`))
			Expect(output).To(ContainSubstring(`"totalJobs"`))
		})

		It("should include issues in JSON output", func() {
			command := exec.Command(pathToCLI, "check", "60 0 * * *", "--json")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring(`"issues"`))
			Expect(output).To(ContainSubstring(`"severity"`))
			Expect(output).To(ContainSubstring(`"code"`))
			Expect(output).To(ContainSubstring(`"CRON-003"`))
		})

		It("should include severity and codes in JSON output with verbose", func() {
			command := exec.Command(pathToCLI, "check", "0 0 1 * 1", "--json", "--verbose")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(2))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring(`"severity"`))
			Expect(output).To(ContainSubstring(`"warn"`))
			Expect(output).To(ContainSubstring(`"code"`))
			Expect(output).To(ContainSubstring(`"CRON-001"`))
			Expect(output).To(ContainSubstring(`"hint"`))
			Expect(output).To(ContainSubstring(`"type"`)) // Backward compatibility
		})
	})

	Context("when running 'cronic check' with alias", func() {
		It("should validate @daily successfully", func() {
			command := exec.Command(pathToCLI, "check", "@daily")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("All valid"))
		})

		It("should validate @hourly successfully", func() {
			command := exec.Command(pathToCLI, "check", "@hourly")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("All valid"))
		})

		It("should validate @weekly successfully", func() {
			command := exec.Command(pathToCLI, "check", "@weekly")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("All valid"))
		})
	})

	Context("when running 'cronic check' with empty crontab", func() {
		It("should handle empty file gracefully", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "empty.cron")
			command := exec.Command(pathToCLI, "check", "--file", testFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("All valid"))
		})
	})

	Context("when running 'cronic check' with various expression types", func() {
		It("should validate step expressions", func() {
			command := exec.Command(pathToCLI, "check", "*/15 * * * *")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("All valid"))
		})

		It("should validate range expressions", func() {
			command := exec.Command(pathToCLI, "check", "0 9-17 * * 1-5")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("All valid"))
		})

		It("should validate list expressions", func() {
			command := exec.Command(pathToCLI, "check", "0 9,12,15 * * *")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("All valid"))
		})
	})

	Context("when running 'cronic check' without arguments", func() {
		It("should attempt to validate user crontab", func() {
			command := exec.Command(pathToCLI, "check")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			// Should exit successfully (even if no crontab exists)
			Eventually(session).Should(gexec.Exit(0))
		})
	})

	Context("when running 'cronic check' with --fail-on flag", func() {
		It("should exit with code 0 for warnings with --fail-on error (default)", func() {
			command := exec.Command(pathToCLI, "check", "0 0 1 * 1")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("All valid"))
		})

		It("should exit with code 2 for warnings with --fail-on warn", func() {
			command := exec.Command(pathToCLI, "check", "0 0 1 * 1", "--fail-on", "warn", "--verbose")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(2))
			Expect(session.Out).To(gbytes.Say("warning"))
		})

		It("should exit with code 1 for errors even with --fail-on warn", func() {
			command := exec.Command(pathToCLI, "check", "60 0 * * *", "--fail-on", "warn")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Out).To(gbytes.Say("issue"))
		})

		It("should show error for invalid --fail-on value", func() {
			command := exec.Command(pathToCLI, "check", "0 0 * * *", "--fail-on", "invalid")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("invalid --fail-on value"))
		})

		It("should work with --fail-on and --json", func() {
			command := exec.Command(pathToCLI, "check", "0 0 1 * 1", "--fail-on", "warn", "--json", "--verbose")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(2))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring(`"severity"`))
			Expect(output).To(ContainSubstring(`"warn"`))
		})
	})

	Context("when running 'cronic check' with --group-by flag", func() {
		It("should group issues by severity", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "mixed_issues.cron")
			// Create a test file with mixed issues if it doesn't exist
			if _, err := os.Stat(testFile); os.IsNotExist(err) {
				// Skip if file doesn't exist, we'll test with inline expression
				command := exec.Command(pathToCLI, "check", "0 0 1 * 1", "--verbose", "--group-by", "severity")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(2))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("warn Issues"))
			}
		})

		It("should group issues by line when using --file", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "invalid.cron")
			command := exec.Command(pathToCLI, "check", "--file", testFile, "--group-by", "line")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring("Line"))
			Expect(output).To(ContainSubstring("━━━"))
		})

		It("should work with --group-by and --json", func() {
			command := exec.Command(pathToCLI, "check", "0 0 1 * 1", "--json", "--verbose", "--group-by", "severity")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(2))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring(`"severity"`))
			Expect(output).To(ContainSubstring(`"warn"`))
		})

		It("should use flat display with --group-by none", func() {
			command := exec.Command(pathToCLI, "check", "60 0 * * *", "--group-by", "none")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring("issue"))
			Expect(output).NotTo(ContainSubstring("━━━"))
		})
	})
})
