package integration_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("CLI Integration Tests", func() {

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

	Describe("Help Command", func() {
		Context("when running 'cronic --help'", func() {
			It("should display help information", func() {
				command := exec.Command(pathToCLI, "--help")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("Available Commands"))
				Expect(output).To(ContainSubstring("version"))
				Expect(output).To(ContainSubstring("explain"))
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
