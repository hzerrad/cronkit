package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Diff Command", func() {
	var (
		oldFile string
		newFile string
	)

	BeforeEach(func() {
		tmpDir := GinkgoT().TempDir()
		oldFile = filepath.Join(tmpDir, "old.cron")
		newFile = filepath.Join(tmpDir, "new.cron")
	})

	AfterEach(func() {
		// Cleanup is handled by GinkgoT().TempDir()
	})

	Context("when comparing two files", func() {
		It("should show added jobs", func() {
			oldContent := "0 2 * * * /usr/bin/backup.sh\n"
			newContent := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"

			err := os.WriteFile(oldFile, []byte(oldContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(newFile, []byte(newContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToCLI, "diff", oldFile, newFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).To(ContainSubstring("Added Jobs"))
			Expect(session.Out.Contents()).To(ContainSubstring("*/15 * * * *"))
		})

		It("should show removed jobs", func() {
			oldContent := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"
			newContent := "0 2 * * * /usr/bin/backup.sh\n"

			err := os.WriteFile(oldFile, []byte(oldContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(newFile, []byte(newContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToCLI, "diff", oldFile, newFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).To(ContainSubstring("Removed Jobs"))
			Expect(session.Out.Contents()).To(ContainSubstring("*/15 * * * *"))
		})

		It("should show modified jobs", func() {
			oldContent := "0 2 * * * /usr/bin/backup.sh # Old comment\n"
			newContent := "0 2 * * * /usr/bin/backup.sh # New comment\n"

			err := os.WriteFile(oldFile, []byte(oldContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(newFile, []byte(newContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToCLI, "diff", oldFile, newFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).To(ContainSubstring("Modified Jobs"))
		})

		It("should output JSON format", func() {
			oldContent := "0 2 * * * /usr/bin/backup.sh\n"
			newContent := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"

			err := os.WriteFile(oldFile, []byte(oldContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(newFile, []byte(newContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToCLI, "diff", oldFile, newFile, "--json")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).To(ContainSubstring(`"added"`))
		})

		It("should output unified format", func() {
			oldContent := "0 2 * * * /usr/bin/backup.sh\n"
			newContent := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"

			err := os.WriteFile(oldFile, []byte(oldContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(newFile, []byte(newContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToCLI, "diff", oldFile, newFile, "--format", "unified")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).To(ContainSubstring("--- old crontab"))
			Expect(session.Out.Contents()).To(ContainSubstring("+++ new crontab"))
		})

		It("should handle identical files", func() {
			content := "0 2 * * * /usr/bin/backup.sh\n"

			err := os.WriteFile(oldFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(newFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToCLI, "diff", oldFile, newFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).To(ContainSubstring("No changes detected"))
		})
	})

	Context("when comparing descriptor schedules", func() {
		It("should show @every and @reboot as added without panicking", func() {
			oldContent := "0 2 * * * /usr/bin/backup.sh\n"
			newContent := "0 2 * * * /usr/bin/backup.sh\n@every 1h /usr/bin/poll.sh\n@reboot /usr/bin/on-boot.sh\n"

			err := os.WriteFile(oldFile, []byte(oldContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(newFile, []byte(newContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToCLI, "diff", oldFile, newFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring("Added Jobs"))
			Expect(output).To(ContainSubstring("@every 1h"))
			Expect(output).To(ContainSubstring("@reboot"))
		})

		It("should show a changed @every interval as modified without panicking", func() {
			oldContent := "@every 1h /usr/bin/poll.sh # steady poll\n"
			newContent := "@every 1h /usr/bin/poll.sh # faster poll\n"

			err := os.WriteFile(oldFile, []byte(oldContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(newFile, []byte(newContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToCLI, "diff", oldFile, newFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).To(ContainSubstring("Modified Jobs"))
		})

		It("should output JSON for a crontab mixing @every and @reboot without panicking", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "descriptors.cron")
			command := exec.Command(pathToCLI, "diff", testFile, testFile, "--json")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).To(ContainSubstring(`"added"`))
		})
	})

	Context("when using flags", func() {
		It("should work with --old-file and --new-file", func() {
			oldContent := "0 2 * * * /usr/bin/backup.sh\n"
			newContent := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"

			err := os.WriteFile(oldFile, []byte(oldContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(newFile, []byte(newContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToCLI, "diff", "--old-file", oldFile, "--new-file", newFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).To(ContainSubstring("Added Jobs"))
		})
	})
})
