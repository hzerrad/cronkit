package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Doc Command", func() {
	var testDataDir string
	var tempDir string

	BeforeEach(func() {
		testDataDir = filepath.Join("..", "..", "testdata", "crontab", "valid")
		var err error
		tempDir, err = os.MkdirTemp("", "cronkit-doc-test-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if tempDir != "" {
			_ = os.RemoveAll(tempDir)
		}
	})

	Describe("Generating documentation", func() {
		Context("when generating markdown documentation", func() {
			It("should generate markdown from a crontab file", func() {
				testFile := filepath.Join(testDataDir, "sample.cron")
				outputFile := filepath.Join(tempDir, "output.md")
				command := exec.Command(pathToCLI, "doc", "--file", testFile, "--output", outputFile, "--format", "md")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Eventually(outputFile).Should(BeAnExistingFile())

				content, err := os.ReadFile(outputFile)
				Expect(err).NotTo(HaveOccurred())
				output := string(content)
				Expect(output).To(ContainSubstring("# Crontab Documentation"))
				Expect(output).To(ContainSubstring("## Summary"))
				Expect(output).To(ContainSubstring("## Jobs"))
			})

			It("should include next runs when requested", func() {
				testFile := filepath.Join(testDataDir, "sample.cron")
				outputFile := filepath.Join(tempDir, "output.md")
				command := exec.Command(pathToCLI, "doc", "--file", testFile, "--output", outputFile, "--format", "md", "--include-next", "5")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Eventually(outputFile).Should(BeAnExistingFile())
				content, err := os.ReadFile(outputFile)
				Expect(err).NotTo(HaveOccurred())
				output := string(content)
				Expect(output).To(ContainSubstring("Next Runs"))
			})

			It("should include statistics when requested", func() {
				testFile := filepath.Join(testDataDir, "sample.cron")
				outputFile := filepath.Join(tempDir, "output.md")
				command := exec.Command(pathToCLI, "doc", "--file", testFile, "--output", outputFile, "--format", "md", "--include-stats")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Eventually(outputFile).Should(BeAnExistingFile())
				content, err := os.ReadFile(outputFile)
				Expect(err).NotTo(HaveOccurred())
				output := string(content)
				Expect(output).To(ContainSubstring("Statistics"))
				Expect(output).To(ContainSubstring("Runs per day"))
			})
		})

		Context("when generating HTML documentation", func() {
			It("should generate HTML from a crontab file", func() {
				testFile := filepath.Join(testDataDir, "sample.cron")
				outputFile := filepath.Join(tempDir, "output.html")
				command := exec.Command(pathToCLI, "doc", "--file", testFile, "--output", outputFile, "--format", "html")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(outputFile).To(BeAnExistingFile())

				content, err := os.ReadFile(outputFile)
				Expect(err).NotTo(HaveOccurred())
				output := string(content)
				Expect(output).To(ContainSubstring("<!DOCTYPE html>"))
				Expect(output).To(ContainSubstring("<h1>"))
				Expect(output).To(ContainSubstring("<table>"))
			})
		})

		Context("when generating JSON documentation", func() {
			It("should generate JSON from a crontab file", func() {
				testFile := filepath.Join(testDataDir, "sample.cron")
				outputFile := filepath.Join(tempDir, "output.json")
				command := exec.Command(pathToCLI, "doc", "--file", testFile, "--output", outputFile, "--format", "json")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(outputFile).To(BeAnExistingFile())

				content, err := os.ReadFile(outputFile)
				Expect(err).NotTo(HaveOccurred())

				var doc map[string]interface{}
				err = json.Unmarshal(content, &doc)
				Expect(err).NotTo(HaveOccurred())
				Expect(doc).To(HaveKey("Title"))
				Expect(doc).To(HaveKey("Jobs"))
				Expect(doc).To(HaveKey("Metadata"))
			})
		})

		Context("when outputting to stdout", func() {
			It("should output markdown to stdout when no output file specified", func() {
				testFile := filepath.Join(testDataDir, "sample.cron")
				command := exec.Command(pathToCLI, "doc", "--file", testFile, "--format", "md")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("# Crontab Documentation"))
			})
		})

		Context("when reading from stdin", func() {
			It("should generate documentation from stdin input", func() {
				crontabContent := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"
				command := exec.Command(pathToCLI, "doc", "--stdin", "--format", "md")
				command.Stdin = strings.NewReader(crontabContent)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("# Crontab Documentation"))
				Expect(output).To(ContainSubstring("backup.sh"))
				Expect(output).To(ContainSubstring("check.sh"))
			})
		})

		Context("when handling invalid crontab", func() {
			It("should handle invalid expressions gracefully", func() {
				crontabContent := "60 0 * * * /usr/bin/invalid.sh\n"
				command := exec.Command(pathToCLI, "doc", "--stdin", "--format", "md")
				command.Stdin = strings.NewReader(crontabContent)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				// Documentation should still be generated, may show invalid jobs
				Expect(output).To(ContainSubstring("# Crontab Documentation"))
			})
		})
	})
})
