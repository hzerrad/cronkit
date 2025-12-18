package integration_test

import (
	"encoding/json"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var pathToCLI string

var _ = BeforeSuite(func() {
	var err error
	pathToCLI, err = gexec.Build("github.com/hzerrad/cronic/cmd/cronic")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

var _ = Describe("Explain Command", func() {

	Describe("Standard Cron Expressions", func() {
		Context("when user explains simple time intervals", func() {
			It("should explain every minute pattern", func() {
				command := exec.Command(pathToCLI, "explain", "* * * * *")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Every minute"))
			})

			It("should explain minute intervals", func() {
				command := exec.Command(pathToCLI, "explain", "*/15 * * * *")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Every 15 minutes"))
			})

			It("should explain hourly pattern", func() {
				command := exec.Command(pathToCLI, "explain", "0 * * * *")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("At the start of every hour"))
			})
		})

		Context("when user explains specific times", func() {
			It("should explain midnight pattern", func() {
				command := exec.Command(pathToCLI, "explain", "0 0 * * *")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("At midnight"))
			})

			It("should explain specific time with 24-hour format", func() {
				command := exec.Command(pathToCLI, "explain", "30 14 * * *")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("At 14:30"))
			})

			It("should explain multiple times per day", func() {
				command := exec.Command(pathToCLI, "explain", "0 9,12,17 * * *")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("At 09:00, 12:00, and 17:00"))
			})
		})

		Context("when user explains day-based patterns", func() {
			It("should explain weekday pattern", func() {
				command := exec.Command(pathToCLI, "explain", "0 9 * * 1-5")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("weekdays"))
			})

			It("should explain specific day of week", func() {
				command := exec.Command(pathToCLI, "explain", "0 0 * * 0")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Sunday"))
			})

			It("should explain specific days of week list", func() {
				command := exec.Command(pathToCLI, "explain", "0 9 * * 1,3,5")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Monday"))
				Expect(session.Out).To(gbytes.Say("Wednesday"))
				Expect(session.Out).To(gbytes.Say("Friday"))
			})

			It("should explain first day of month", func() {
				command := exec.Command(pathToCLI, "explain", "0 0 1 * *")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("first day of every month"))
			})
		})

		Context("when user explains complex business patterns", func() {
			It("should explain business hours interval pattern", func() {
				command := exec.Command(pathToCLI, "explain", "*/5 9-17 * * 1-5")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Every 5 minutes"))
				Expect(session.Out).To(gbytes.Say("between 09:00 and 17:59"))
				Expect(session.Out).To(gbytes.Say("weekdays"))
			})
		})
	})

	Describe("Cron Aliases", func() {
		Context("when user uses standard aliases", func() {
			It("should explain @daily alias", func() {
				command := exec.Command(pathToCLI, "explain", "@daily")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("At midnight every day"))
			})

			It("should explain @hourly alias", func() {
				command := exec.Command(pathToCLI, "explain", "@hourly")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("At the start of every hour"))
			})

			It("should explain @weekly alias", func() {
				command := exec.Command(pathToCLI, "explain", "@weekly")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("At midnight every Sunday"))
			})

			It("should explain @monthly alias", func() {
				command := exec.Command(pathToCLI, "explain", "@monthly")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("At midnight on the first day of every month"))
			})

			It("should explain @yearly alias", func() {
				command := exec.Command(pathToCLI, "explain", "@yearly")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("At midnight on January 1st"))
			})
		})
	})

	Describe("JSON Output", func() {
		Context("when user requests JSON format", func() {
			It("should output valid JSON with --json flag", func() {
				command := exec.Command(pathToCLI, "explain", "0 0 * * *", "--json")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))

				var result map[string]string
				err = json.Unmarshal(session.Out.Contents(), &result)
				Expect(err).NotTo(HaveOccurred())

				Expect(result).To(HaveKey("expression"))
				Expect(result).To(HaveKey("description"))
				Expect(result["expression"]).To(Equal("0 0 * * *"))
				Expect(result["description"]).To(ContainSubstring("midnight"))
			})

			It("should output valid JSON with -j flag", func() {
				command := exec.Command(pathToCLI, "explain", "@daily", "-j")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))

				var result map[string]string
				err = json.Unmarshal(session.Out.Contents(), &result)
				Expect(err).NotTo(HaveOccurred())

				Expect(result["expression"]).To(Equal("@daily"))
				Expect(result["description"]).NotTo(BeEmpty())
			})

			It("should include original expression in JSON output", func() {
				command := exec.Command(pathToCLI, "explain", "*/15 * * * *", "--json")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))

				var result map[string]string
				err = json.Unmarshal(session.Out.Contents(), &result)
				Expect(err).NotTo(HaveOccurred())

				Expect(result["expression"]).To(Equal("*/15 * * * *"))
				Expect(result["description"]).To(ContainSubstring("15 minutes"))
			})
		})
	})

	Describe("Error Handling", func() {
		Context("when user provides invalid input", func() {
			It("should reject expressions with wrong field count", func() {
				command := exec.Command(pathToCLI, "explain", "0 0 *")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("expected 5 fields"))
			})

			It("should reject out of range values", func() {
				command := exec.Command(pathToCLI, "explain", "60 0 * * *")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("out of range"))
			})

			It("should reject completely invalid expressions", func() {
				command := exec.Command(pathToCLI, "explain", "not-a-cron-expression")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("failed to parse"))
			})

			It("should require an argument", func() {
				command := exec.Command(pathToCLI, "explain")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
			})
		})

		Context("when user needs help", func() {
			It("should provide help with --help flag", func() {
				command := exec.Command(pathToCLI, "explain", "--help")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("Usage:"))
				Expect(output).To(ContainSubstring("Examples:"))
			})

			It("should show examples in help text", func() {
				command := exec.Command(pathToCLI, "explain", "-h")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("cronic explain"))
				Expect(output).To(ContainSubstring("@daily"))
			})
		})
	})

	Describe("User Workflows", func() {
		Context("when DevOps engineer sets up monitoring", func() {
			It("should help understand existing crontab entries", func() {
				By("checking a backup schedule")
				command := exec.Command(pathToCLI, "explain", "0 2 * * *")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("At 02:00"))

				By("checking a monitoring interval")
				command = exec.Command(pathToCLI, "explain", "*/5 * * * *")
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Every 5 minutes"))
			})
		})

		Context("when developer creates scheduled jobs", func() {
			It("should verify business hours schedule", func() {
				command := exec.Command(pathToCLI, "explain", "*/10 8-18 * * 1-5")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Every 10 minutes"))
				Expect(session.Out).To(gbytes.Say("08:00"))
				Expect(session.Out).To(gbytes.Say("weekdays"))
			})
		})
	})
})
