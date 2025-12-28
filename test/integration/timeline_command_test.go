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

var _ = Describe("Timeline Command", func() {
	Context("when running 'cronic timeline' with a valid expression", func() {
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

	Context("when running 'cronic timeline' with --json flag", func() {
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

	Context("when running 'cronic timeline' with --file flag", func() {
		It("should read from crontab file", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "sample.cron")
			command := exec.Command(pathToCLI, "timeline", "--file", testFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Timeline"))
		})

		It("should output JSON for crontab file", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "sample.cron")
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

	Context("when running 'cronic timeline' with invalid expression", func() {
		It("should exit with error code 1", func() {
			command := exec.Command(pathToCLI, "timeline", "60 0 * * *")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("invalid"))
		})
	})

	Context("when running 'cronic timeline' with invalid view", func() {
		It("should exit with error", func() {
			command := exec.Command(pathToCLI, "timeline", "*/15 * * * *", "--view", "invalid")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("invalid view type"))
		})
	})

	Context("when running 'cronic timeline' with non-existent file", func() {
		It("should exit with error code 1", func() {
			command := exec.Command(pathToCLI, "timeline", "--file", "/nonexistent/file.cron")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("failed to read"))
		})
	})

	Context("when running 'cronic timeline' without arguments", func() {
		It("should attempt to read user crontab", func() {
			command := exec.Command(pathToCLI, "timeline")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			// May exit with 0 (empty crontab) or 1 (error reading), but should not crash
			Eventually(session).Should(gexec.Exit())
		})
	})
})
