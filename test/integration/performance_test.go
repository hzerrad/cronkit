package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Performance Tests", func() {
	var pathToCLI string

	BeforeSuite(func() {
		var err error
		pathToCLI, err = gexec.Build("github.com/hzerrad/cronic/cmd/cronic")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterSuite(func() {
		gexec.CleanupBuildArtifacts()
	})

	Context("when processing large crontabs", func() {
		It("should process 100 jobs in under 1 second", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "performance", "large.cron")
			Expect(testFile).To(BeAnExistingFile())

			start := time.Now()
			command := exec.Command(pathToCLI, "check", "--file", testFile)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			duration := time.Since(start)

			Expect(duration).To(BeNumerically("<", 1*time.Second),
				"Processing 100 jobs should take less than 1 second, took %v", duration)
		})

		It("should list 100 jobs in under 1 second", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "performance", "large.cron")
			Expect(testFile).To(BeAnExistingFile())

			start := time.Now()
			command := exec.Command(pathToCLI, "list", "--file", testFile, "--json")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			duration := time.Since(start)

			Expect(duration).To(BeNumerically("<", 1*time.Second),
				"Listing 100 jobs should take less than 1 second, took %v", duration)
		})

		It("should validate 100 jobs in under 1 second", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "performance", "large.cron")
			Expect(testFile).To(BeAnExistingFile())

			start := time.Now()
			command := exec.Command(pathToCLI, "check", "--file", testFile, "--verbose")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			duration := time.Since(start)

			Expect(duration).To(BeNumerically("<", 1*time.Second),
				"Validating 100 jobs should take less than 1 second, took %v", duration)
		})

		It("should process 500 jobs in under 5 seconds", func() {
			// Create a temporary file with 500 jobs
			tmpFile, err := os.CreateTemp("", "perf-test-500-*.cron")
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				_ = os.Remove(tmpFile.Name())
			}()

			// Write 500 jobs to the file
			for i := 0; i < 500; i++ {
				_, err := tmpFile.WriteString("0 * * * * /usr/bin/job" + string(rune('0'+(i%10))) + ".sh\n")
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(tmpFile.Close()).To(Succeed())

			start := time.Now()
			command := exec.Command(pathToCLI, "check", "--file", tmpFile.Name())
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			duration := time.Since(start)

			Expect(duration).To(BeNumerically("<", 5*time.Second),
				"Processing 500 jobs should take less than 5 seconds, took %v", duration)
		})
	})

	Context("when processing timeline with many jobs", func() {
		It("should generate timeline for 100 jobs in reasonable time", func() {
			testFile := filepath.Join("..", "..", "testdata", "crontab", "performance", "large.cron")
			Expect(testFile).To(BeAnExistingFile())

			start := time.Now()
			command := exec.Command(pathToCLI, "timeline", "--file", testFile, "--json")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(0))
			duration := time.Since(start)

			// Timeline generation is more complex, allow up to 3 seconds
			Expect(duration).To(BeNumerically("<", 3*time.Second),
				"Generating timeline for 100 jobs should take less than 3 seconds, took %v", duration)
		})
	})
})
