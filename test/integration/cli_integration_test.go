package integration_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("CLI Integration Tests", func() {
	var (
		pathToCLI string
	)

	BeforeSuite(func() {
		var err error
		// Build the CLI binary for testing
		pathToCLI, err = gexec.Build("github.com/hzerrad/cronic/cmd/cronic")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterSuite(func() {
		// Clean up the built binary
		gexec.CleanupBuildArtifacts()
	})

	Describe("Version Command", func() {
		Context("when running 'cronic version'", func() {
			It("should display version information", func() {
				command := exec.Command(pathToCLI, "version")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("cronic"))
			})
		})

		Context("when running 'cronic --version'", func() {
			It("should display version information", func() {
				command := exec.Command(pathToCLI, "--version")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("cronic"))
			})
		})
	})

	Describe("Example Command", func() {
		Context("when running 'cronic example' without flags", func() {
			It("should display default greeting", func() {
				command := exec.Command(pathToCLI, "example")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Hello from cronic!"))
			})
		})

		Context("when running 'cronic example --name World'", func() {
			It("should display personalized greeting", func() {
				command := exec.Command(pathToCLI, "example", "--name", "World")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Hello, World!"))
			})
		})

		Context("when running 'cronic example -n TestUser'", func() {
			It("should display personalized greeting with short flag", func() {
				command := exec.Command(pathToCLI, "example", "-n", "TestUser")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Hello, TestUser!"))
			})
		})
	})

	Describe("Help Command", func() {
		Context("when running 'cronic --help'", func() {
			It("should display help information", func() {
				command := exec.Command(pathToCLI, "--help")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Available Commands"))
				Expect(session.Out).To(gbytes.Say("version"))
				Expect(session.Out).To(gbytes.Say("example"))
			})
		})

		Context("when running 'cronic help version'", func() {
			It("should display help for version command", func() {
				command := exec.Command(pathToCLI, "help", "version")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("version"))
			})
		})
	})

	Describe("Invalid Command", func() {
		Context("when running an unknown command", func() {
			It("should return an error", func() {
				command := exec.Command(pathToCLI, "nonexistent")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("unknown command"))
			})
		})
	})
})
