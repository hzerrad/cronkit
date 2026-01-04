package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Budget Command", func() {
	var testFile string

	BeforeEach(func() {
		tmpDir := GinkgoT().TempDir()
		testFile = filepath.Join(tmpDir, "test.cron")
	})

	Context("when analyzing crontab", func() {
		It("should pass when budget is met", func() {
			content := "0 * * * * /usr/bin/job1.sh\n15 * * * * /usr/bin/job2.sh\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToCLI, "budget", "--file", testFile, "--max-concurrent", "10", "--window", "1h")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).To(ContainSubstring("Budget Analysis"))
			Expect(session.Out.Contents()).To(ContainSubstring("PASSED"))
		})

		It("should fail when budget is violated", func() {
			// Create jobs that will overlap (all run at minute 0)
			content := "0 * * * * /usr/bin/job1.sh\n0 * * * * /usr/bin/job2.sh\n0 * * * * /usr/bin/job3.sh\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToCLI, "budget", "--file", testFile, "--max-concurrent", "2", "--window", "1h")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0)) // Report-only mode, exit 0
			Expect(session.Out.Contents()).To(ContainSubstring("Budget Analysis"))
		})

		It("should enforce budget with --enforce flag", func() {
			content := "0 * * * * /usr/bin/job1.sh\n0 * * * * /usr/bin/job2.sh\n0 * * * * /usr/bin/job3.sh\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToCLI, "budget", "--file", testFile, "--max-concurrent", "2", "--window", "1h", "--enforce")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			// Should exit with error when budget is violated
			Eventually(session).Should(gexec.Exit(1))
		})

		It("should output JSON format", func() {
			content := "0 * * * * /usr/bin/job1.sh\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToCLI, "budget", "--file", testFile, "--max-concurrent", "10", "--window", "1h", "--json")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).To(ContainSubstring(`"passed"`))
			Expect(session.Out.Contents()).To(ContainSubstring(`"budgets"`))
		})

		It("should show verbose output", func() {
			content := "0 * * * * /usr/bin/job1.sh\n0 * * * * /usr/bin/job2.sh\n0 * * * * /usr/bin/job3.sh\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToCLI, "budget", "--file", testFile, "--max-concurrent", "2", "--window", "1h", "--verbose")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring("Budget Analysis"))
		})

		It("should read from stdin", func() {
			content := "0 * * * * /usr/bin/job1.sh\n"
			command := exec.Command(pathToCLI, "budget", "--stdin", "--max-concurrent", "10", "--window", "1h")
			command.Stdin = strings.NewReader(content)

			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).To(ContainSubstring("Budget Analysis"))
		})

		It("should error when max-concurrent is missing", func() {
			command := exec.Command(pathToCLI, "budget", "--file", testFile, "--window", "1h")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err.Contents()).To(ContainSubstring("max-concurrent"))
		})

		It("should error when window is missing", func() {
			command := exec.Command(pathToCLI, "budget", "--file", testFile, "--max-concurrent", "10")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err.Contents()).To(ContainSubstring("window"))
		})

		It("should error when window is invalid", func() {
			content := "0 * * * * /usr/bin/job1.sh\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToCLI, "budget", "--file", testFile, "--max-concurrent", "10", "--window", "invalid")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err.Contents()).To(ContainSubstring("invalid"))
		})
	})
})
