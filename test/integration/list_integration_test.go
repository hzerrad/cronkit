package integration_test

import (
	"encoding/json"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("List Command", func() {
	var testDataDir string

	BeforeEach(func() {
		testDataDir = filepath.Join("..", "..", "testdata", "crontab", "valid")
	})

	Describe("Listing crontab files", func() {
		Context("when listing a valid crontab file", func() {
			It("should display all jobs in a table format", func() {
				sampleFile := filepath.Join(testDataDir, "sample.cron")
				command := exec.Command(pathToCLI, "list", "--file", sampleFile)
				session, err := gexec.Start(command, nil, nil)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())

				// Should contain header
				Expect(output).To(ContainSubstring("LINE"))
				Expect(output).To(ContainSubstring("EXPRESSION"))
				Expect(output).To(ContainSubstring("DESCRIPTION"))
				Expect(output).To(ContainSubstring("COMMAND"))

				// Should contain job details
				Expect(output).To(ContainSubstring("backup"))
				Expect(output).To(ContainSubstring("check-disk"))
			})

			It("should include humanized descriptions", func() {
				sampleFile := filepath.Join(testDataDir, "sample.cron")
				command := exec.Command(pathToCLI, "list", "--file", sampleFile)
				session, err := gexec.Start(command, nil, nil)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())

				// Should contain humanized time descriptions
				Expect(output).To(MatchRegexp("At.*02:00"))
				Expect(output).To(ContainSubstring("Every 15 minutes"))
			})
		})

		Context("when listing with --json flag", func() {
			It("should output valid JSON", func() {
				sampleFile := filepath.Join(testDataDir, "sample.cron")
				command := exec.Command(pathToCLI, "list", "--file", sampleFile, "--json")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := session.Out.Contents()

				// Should be valid JSON
				var result map[string]interface{}
				err = json.Unmarshal(output, &result)
				Expect(err).NotTo(HaveOccurred())

				// Should have jobs array
				jobs, ok := result["jobs"].([]interface{})
				Expect(ok).To(BeTrue())
				Expect(jobs).NotTo(BeEmpty())

				// First job should have expected fields
				firstJob := jobs[0].(map[string]interface{})
				Expect(firstJob).To(HaveKey("lineNumber"))
				Expect(firstJob).To(HaveKey("expression"))
				Expect(firstJob).To(HaveKey("command"))
				Expect(firstJob).To(HaveKey("description"))
			})

			It("should include humanized descriptions in JSON", func() {
				sampleFile := filepath.Join(testDataDir, "sample.cron")
				command := exec.Command(pathToCLI, "list", "--file", sampleFile, "--json")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := session.Out.Contents()

				var result map[string]interface{}
				err = json.Unmarshal(output, &result)
				Expect(err).NotTo(HaveOccurred())

				jobs := result["jobs"].([]interface{})
				for _, job := range jobs {
					jobMap := job.(map[string]interface{})
					description, hasDesc := jobMap["description"]
					if hasDesc {
						Expect(description).NotTo(BeEmpty())
					}
				}
			})
		})

		Context("when listing with --all flag", func() {
			It("should include comments and environment variables", func() {
				sampleFile := filepath.Join(testDataDir, "sample.cron")
				command := exec.Command(pathToCLI, "list", "--file", sampleFile, "--all")
				session, err := gexec.Start(command, nil, nil)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())

				// Should contain environment variables
				Expect(output).To(ContainSubstring("SHELL"))
				Expect(output).To(ContainSubstring("PATH"))
				Expect(output).To(ContainSubstring("MAILTO"))

				// Should contain entry type indicators
				Expect(output).To(ContainSubstring("ENV"))
				Expect(output).To(ContainSubstring("JOB"))
			})

			It("should output all entries in JSON format with --all and --json", func() {
				sampleFile := filepath.Join(testDataDir, "sample.cron")
				command := exec.Command(pathToCLI, "list", "--file", sampleFile, "--all", "--json")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := session.Out.Contents()

				var result map[string]interface{}
				err = json.Unmarshal(output, &result)
				Expect(err).NotTo(HaveOccurred())

				entries, ok := result["entries"].([]interface{})
				Expect(ok).To(BeTrue())
				Expect(entries).NotTo(BeEmpty())

				// Should have different entry types
				hasEnvVar := false
				hasJob := false
				for _, entry := range entries {
					entryMap := entry.(map[string]interface{})
					entryType := entryMap["type"].(string)
					if entryType == "ENV" {
						hasEnvVar = true
					}
					if entryType == "JOB" {
						hasJob = true
					}
				}
				Expect(hasEnvVar).To(BeTrue())
				Expect(hasJob).To(BeTrue())
			})
		})

		Context("when listing an empty crontab file", func() {
			It("should display a 'no jobs found' message", func() {
				emptyFile := filepath.Join(testDataDir, "empty.cron")
				command := exec.Command(pathToCLI, "list", "--file", emptyFile)
				session, err := gexec.Start(command, nil, nil)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("No cron jobs found"))
			})

			It("should output empty jobs array in JSON", func() {
				emptyFile := filepath.Join(testDataDir, "empty.cron")
				command := exec.Command(pathToCLI, "list", "--file", emptyFile, "--json")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := session.Out.Contents()

				var result map[string]interface{}
				err = json.Unmarshal(output, &result)
				Expect(err).NotTo(HaveOccurred())

				jobs, ok := result["jobs"].([]interface{})
				Expect(ok).To(BeTrue())
				Expect(jobs).To(BeEmpty())
			})
		})

		Context("when file does not exist", func() {
			It("should return an error", func() {
				command := exec.Command(pathToCLI, "list", "--file", "/nonexistent/file.cron")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("failed to read crontab"))
			})
		})

		Context("when listing with invalid cron entries", func() {
			It("should skip invalid entries and show valid ones", func() {
				invalidFile := filepath.Join("..", "..", "testdata", "crontab", "invalid", "invalid.cron")
				command := exec.Command(pathToCLI, "list", "--file", invalidFile)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				// Should succeed (shows what it can parse)
				Eventually(session).Should(gexec.Exit(0))
			})
		})
	})

	Describe("Help and usage", func() {
		Context("when running 'cronkit list --help'", func() {
			It("should display help information", func() {
				command := exec.Command(pathToCLI, "list", "--help")
				session, err := gexec.Start(command, nil, nil)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())

				Expect(output).To(ContainSubstring("list"))
				Expect(output).To(ContainSubstring("--file"))
				Expect(output).To(ContainSubstring("--all"))
				Expect(output).To(ContainSubstring("--json"))
			})
		})
	})

	Describe("Alias jobs", func() {
		Context("when listing crontab with @-aliases", func() {
			It("should parse and humanize alias expressions", func() {
				sampleFile := filepath.Join(testDataDir, "sample.cron")
				command := exec.Command(pathToCLI, "list", "--file", sampleFile)
				session, err := gexec.Start(command, nil, nil)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())

				// Should contain aliases
				Expect(output).To(ContainSubstring("@monthly"))
				Expect(output).To(ContainSubstring("@hourly"))
			})
		})
	})

	Describe("Locale support", func() {
		Context("when parsing crontab with day names", func() {
			It("should correctly parse MON/TUE/etc symbols", func() {
				sampleFile := filepath.Join(testDataDir, "sample.cron")
				command := exec.Command(pathToCLI, "list", "--file", sampleFile, "--locale", "en")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				// Should not error on day name parsing
			})
		})
	})
})
