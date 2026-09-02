package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

// scanFixture copies testdata/scan into a temp dir with its own ".git" marker
// so discover.FindRoot anchors there instead of this checkout's real root.
func scanFixture() string {
	dir := GinkgoT().TempDir()
	src := filepath.Join("..", "..", "testdata", "scan")
	Expect(os.CopyFS(dir, os.DirFS(src))).To(Succeed())
	Expect(os.MkdirAll(filepath.Join(dir, ".git"), 0o755)).To(Succeed())
	return dir
}

var _ = Describe("Scan Command", func() {
	var fixtureDir string

	BeforeEach(func() {
		fixtureDir = scanFixture()
	})

	Describe("scanning a fixture repository", func() {
		Context("with the default table output", func() {
			It("finds schedules from more than one source and reports the fixture's broken manifest on stderr", func() {
				command := exec.Command(pathToCLI, "scan", fixtureDir)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())

				Expect(output).To(ContainSubstring("SOURCE"))
				Expect(output).To(ContainSubstring("EXPRESSION"))
				Expect(output).To(ContainSubstring("crontab"))
				Expect(output).To(ContainSubstring("k8s"))
				Expect(output).To(ContainSubstring("argo"))
				Expect(output).To(ContainSubstring("gha"))
				Expect(output).To(ContainSubstring("schedule(s) across"))

				// Problems are diagnostics, not results: they belong on
				// stderr, never mixed into the table on stdout.
				Expect(session.Err).To(gbytes.Say("broken.yaml"))
				Expect(output).NotTo(ContainSubstring("broken.yaml"))
			})
		})

		Context("with --json", func() {
			It("decodes cleanly and carries an item per schedule from every source", func() {
				command := exec.Command(pathToCLI, "scan", fixtureDir, "--json")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))

				var inv map[string]interface{}
				Expect(json.Unmarshal(session.Out.Contents(), &inv)).To(Succeed())
				Expect(inv).To(HaveKeyWithValue("schemaVersion", "1"))
				Expect(inv).To(HaveKeyWithValue("root", fixtureDir))

				items, ok := inv["items"].([]interface{})
				Expect(ok).To(BeTrue())
				Expect(items).NotTo(BeEmpty())

				bySource := map[string]int{}
				for _, raw := range items {
					item, ok := raw.(map[string]interface{})
					Expect(ok).To(BeTrue())
					Expect(item).To(HaveKey("expression"))
					Expect(item).To(HaveKey("locator"))
					source, _ := item["source"].(string)
					bySource[source]++
				}
				Expect(bySource).To(Equal(map[string]int{
					"crontab": 1,
					"k8s":     1,
					"argo":    2,
					"gha":     2,
				}))
			})
		})
	})

	Describe("exit codes", func() {
		Context("when the walk reports a per-file problem but --strict is not set", func() {
			It("still exits 0", func() {
				command := exec.Command(pathToCLI, "scan", fixtureDir, "--json")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Err).To(gbytes.Say("broken.yaml"))
			})
		})

		Context("when the walk reports a per-file problem and --strict is set", func() {
			It("exits 1 while still emitting the inventory it found", func() {
				command := exec.Command(pathToCLI, "scan", fixtureDir, "--json", "--strict")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("broken.yaml"))
				Expect(session.Out.Contents()).NotTo(BeEmpty())
			})
		})

		Context("when --strict is set but the walk reports no problems", func() {
			It("exits 0", func() {
				// Restricting to the crontab source alone never touches the
				// fixture's broken.yaml, so the walk reports nothing.
				command := exec.Command(pathToCLI, "scan", fixtureDir, "--json", "--strict", "--source=crontab")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Err.Contents()).To(BeEmpty())
			})
		})
	})

	Describe("an unknown --source", func() {
		It("fails loudly, naming both the bad id and the valid ones", func() {
			command := exec.Command(pathToCLI, "scan", fixtureDir, "--source=nope")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say(`unknown --source "nope"`))
			Expect(session.Err).To(gbytes.Say("crontab"))
		})
	})

	Describe("determinism", func() {
		It("produces byte-identical output across two consecutive scans", func() {
			run := func() []byte {
				command := exec.Command(pathToCLI, "scan", fixtureDir, "--json")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				return session.Out.Contents()
			}

			first := run()
			second := run()
			Expect(first).To(Equal(second), "two scans of an unchanged tree must emit byte-identical JSON")
		})
	})
})
