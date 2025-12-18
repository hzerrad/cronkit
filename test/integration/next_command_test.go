package integration_test

import (
	"encoding/json"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Next Command", func() {

	Describe("Basic Usage", func() {
		Context("when user calculates next runs", func() {
			It("should show next 10 runs by default", func() {
				command := exec.Command(pathToCLI, "next", "*/15 * * * *")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Next 10 runs"))
				Expect(session.Out).To(gbytes.Say("\\*/15 \\* \\* \\* \\*"))
				Expect(session.Out).To(gbytes.Say("1\\."))
				Expect(session.Out).To(gbytes.Say("10\\."))
			})

			It("should respect custom count with long flag", func() {
				command := exec.Command(pathToCLI, "next", "@daily", "--count", "5")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Next 5 runs"))
				Expect(session.Out).To(gbytes.Say("@daily"))
			})

			It("should respect custom count with short flag", func() {
				command := exec.Command(pathToCLI, "next", "@hourly", "-c", "3")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Next 3 runs"))
			})

			It("should handle count of 1", func() {
				command := exec.Command(pathToCLI, "next", "@daily", "--count", "1")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Next 1 run"))
			})

			It("should handle maximum count of 100", func() {
				command := exec.Command(pathToCLI, "next", "0 * * * *", "-c", "100")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Next 100 runs"))
			})
		})
	})

	Describe("Cron Expressions", func() {
		Context("when user uses interval patterns", func() {
			It("should calculate next runs for minute intervals", func() {
				command := exec.Command(pathToCLI, "next", "*/5 * * * *", "-c", "3")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("Every 5 minutes"))
			})

			It("should calculate next runs for hourly pattern", func() {
				command := exec.Command(pathToCLI, "next", "0 * * * *", "-c", "3")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("hour"))
			})
		})

		Context("when user uses cron aliases", func() {
			It("should handle @daily alias", func() {
				command := exec.Command(pathToCLI, "next", "@daily", "-c", "3")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("@daily"))
				Expect(output).To(ContainSubstring("midnight"))
			})

			It("should handle @weekly alias", func() {
				command := exec.Command(pathToCLI, "next", "@weekly", "-c", "2")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("Sunday"))
			})

			It("should handle @monthly alias", func() {
				command := exec.Command(pathToCLI, "next", "@monthly", "-c", "2")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(strings.ToLower(output)).To(ContainSubstring("first day"))
			})

			It("should handle @yearly alias", func() {
				command := exec.Command(pathToCLI, "next", "@yearly", "-c", "2")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("January"))
			})
		})

		Context("when user uses weekday patterns", func() {
			It("should calculate next runs for weekdays", func() {
				command := exec.Command(pathToCLI, "next", "0 9 * * 1-5", "-c", "5")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(strings.ToLower(output)).To(ContainSubstring("weekdays"))
			})

			It("should calculate next runs for specific weekday", func() {
				command := exec.Command(pathToCLI, "next", "0 0 * * 0", "-c", "2")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("Sunday"))
			})
		})

		Context("when user uses complex business patterns", func() {
			It("should handle business hours pattern", func() {
				command := exec.Command(pathToCLI, "next", "*/5 9-17 * * 1-5", "-c", "10")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("Every 5 minutes"))
			})
		})
	})

	Describe("JSON Output", func() {
		Context("when user requests JSON format", func() {
			It("should output valid JSON with long flag", func() {
				command := exec.Command(pathToCLI, "next", "@daily", "--json", "-c", "3")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))

				var result map[string]interface{}
				err = json.Unmarshal(session.Out.Contents(), &result)
				Expect(err).NotTo(HaveOccurred())

				Expect(result).To(HaveKey("expression"))
				Expect(result).To(HaveKey("description"))
				Expect(result).To(HaveKey("timezone"))
				Expect(result).To(HaveKey("next_runs"))

				Expect(result["expression"]).To(Equal("@daily"))
				Expect(result["description"]).To(ContainSubstring("midnight"))
			})

			It("should output valid JSON with short flag", func() {
				command := exec.Command(pathToCLI, "next", "*/15 * * * *", "-j", "-c", "5")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))

				var result map[string]interface{}
				err = json.Unmarshal(session.Out.Contents(), &result)
				Expect(err).NotTo(HaveOccurred())

				nextRuns := result["next_runs"].([]interface{})
				Expect(len(nextRuns)).To(Equal(5))
			})

			It("should include run details in JSON", func() {
				command := exec.Command(pathToCLI, "next", "@hourly", "--json", "-c", "2")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))

				var result map[string]interface{}
				err = json.Unmarshal(session.Out.Contents(), &result)
				Expect(err).NotTo(HaveOccurred())

				nextRuns := result["next_runs"].([]interface{})
				firstRun := nextRuns[0].(map[string]interface{})

				Expect(firstRun).To(HaveKey("number"))
				Expect(firstRun).To(HaveKey("timestamp"))
				Expect(firstRun).To(HaveKey("relative"))

				// Verify run number
				Expect(firstRun["number"]).To(BeNumerically("==", 1))

				// Verify timestamp format (RFC3339)
				timestamp := firstRun["timestamp"].(string)
				Expect(timestamp).To(MatchRegexp(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`))

				// Verify relative time format
				relative := firstRun["relative"].(string)
				Expect(relative).To(ContainSubstring("in "))
			})

			It("should have sequential run numbers", func() {
				command := exec.Command(pathToCLI, "next", "* * * * *", "--json", "-c", "5")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))

				var result map[string]interface{}
				err = json.Unmarshal(session.Out.Contents(), &result)
				Expect(err).NotTo(HaveOccurred())

				nextRuns := result["next_runs"].([]interface{})
				for i, run := range nextRuns {
					runMap := run.(map[string]interface{})
					Expect(runMap["number"]).To(BeNumerically("==", i+1))
				}
			})
		})
	})

	Describe("Text Output", func() {
		Context("when user views text output", func() {
			It("should show timestamps with timezone", func() {
				command := exec.Command(pathToCLI, "next", "@daily", "-c", "2")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())

				// Should contain timestamp pattern (YYYY-MM-DD HH:MM:SS TZ)
				Expect(output).To(MatchRegexp(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} \w+`))
			})

			It("should show numbered list of runs", func() {
				command := exec.Command(pathToCLI, "next", "0 * * * *", "-c", "3")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())

				Expect(output).To(MatchRegexp(`1\.`))
				Expect(output).To(MatchRegexp(`2\.`))
				Expect(output).To(MatchRegexp(`3\.`))
			})

			It("should include human description", func() {
				command := exec.Command(pathToCLI, "next", "@weekly", "-c", "1")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())

				Expect(output).To(ContainSubstring("Sunday"))
			})
		})
	})

	Describe("Error Handling", func() {
		Context("when user provides invalid input", func() {
			It("should reject expressions with wrong field count", func() {
				command := exec.Command(pathToCLI, "next", "0 0 *")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("expected 5 fields"))
			})

			It("should reject out of range values", func() {
				command := exec.Command(pathToCLI, "next", "60 0 * * *")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("out of range"))
			})

			It("should reject invalid expressions", func() {
				command := exec.Command(pathToCLI, "next", "not-a-cron")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("expected 5 fields"))
			})

			It("should reject invalid alias", func() {
				command := exec.Command(pathToCLI, "next", "@invalid")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("unrecognized descriptor"))
			})

			It("should require an argument", func() {
				command := exec.Command(pathToCLI, "next")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
			})
		})

		Context("when user provides invalid count", func() {
			It("should reject count of 0", func() {
				command := exec.Command(pathToCLI, "next", "* * * * *", "--count", "0")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("count must be at least 1"))
			})

			It("should reject negative count", func() {
				command := exec.Command(pathToCLI, "next", "* * * * *", "-c", "-5")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("count must be at least 1"))
			})

			It("should reject count over 100", func() {
				command := exec.Command(pathToCLI, "next", "* * * * *", "--count", "101")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("count must be at most 100"))
			})
		})
	})

	Describe("Help Documentation", func() {
		Context("when user needs help", func() {
			It("should provide help with long flag", func() {
				command := exec.Command(pathToCLI, "next", "--help")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())

				Expect(output).To(ContainSubstring("Usage:"))
				Expect(output).To(ContainSubstring("next"))
				Expect(output).To(ContainSubstring("Examples:"))
				Expect(output).To(ContainSubstring("--count"))
				Expect(output).To(ContainSubstring("--json"))
			})

			It("should provide help with short flag", func() {
				command := exec.Command(pathToCLI, "next", "-h")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())

				Expect(output).To(ContainSubstring("Usage:"))
				Expect(output).To(ContainSubstring("@daily"))
			})

			It("should show examples in help", func() {
				command := exec.Command(pathToCLI, "help", "next")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())

				Expect(output).To(ContainSubstring("*/15 * * * *"))
				Expect(output).To(ContainSubstring("@daily"))
				Expect(output).To(ContainSubstring("0 9 * * 1-5"))
			})
		})
	})

	Describe("User Workflows", func() {
		Context("when DevOps engineer plans job schedules", func() {
			It("should help verify backup schedule", func() {
				By("checking nightly backup at 2 AM")
				command := exec.Command(pathToCLI, "next", "0 2 * * *", "-c", "7")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("02:00"))
			})

			It("should help plan monitoring intervals", func() {
				By("checking 5-minute monitoring schedule")
				command := exec.Command(pathToCLI, "next", "*/5 * * * *", "-c", "12")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("Next 12 runs"))
			})
		})

		Context("when developer creates scheduled jobs", func() {
			It("should verify business hours schedule", func() {
				command := exec.Command(pathToCLI, "next", "*/10 8-18 * * 1-5", "-c", "10")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("Every 10 minutes"))
				Expect(strings.ToLower(output)).To(ContainSubstring("weekdays"))
			})

			It("should check monthly report schedule", func() {
				command := exec.Command(pathToCLI, "next", "0 0 1 * *", "-c", "6")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(strings.ToLower(output)).To(ContainSubstring("first day"))
			})
		})

		Context("when system administrator debugs cron issues", func() {
			It("should help compare expected vs actual run times", func() {
				By("getting next runs in JSON for programmatic comparison")
				command := exec.Command(pathToCLI, "next", "0 */6 * * *", "--json", "-c", "4")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))

				var result map[string]interface{}
				err = json.Unmarshal(session.Out.Contents(), &result)
				Expect(err).NotTo(HaveOccurred())

				nextRuns := result["next_runs"].([]interface{})
				Expect(len(nextRuns)).To(Equal(4))
			})
		})
	})
})
