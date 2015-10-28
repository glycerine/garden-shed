package aufs_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry-incubator/garden-shed/docker_drivers/aufs"
	"github.com/docker/docker/daemon/graphdriver"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

type QuotaedDriver interface {
	GetQuotaed(id, mountlabel string) (string, error)
}

var _ = Describe("Aufs", func() {
	var (
		driver    *aufs.Driver
		graphRoot string
		id        string
	)

	BeforeEach(func() {
		var err error

		graphRoot, err = ioutil.TempDir("", "aufsGraphRoot")
		Expect(err).NotTo(HaveOccurred())

		ps, err := gexec.Start(
			exec.Command("mount", "-t", "tmpfs", "tmpfs", graphRoot),
			GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(ps).Should(gexec.Exit(0))

		var graphDriver graphdriver.Driver
		graphDriver, err = aufs.Init(filepath.Join(graphRoot, "aufs"), []string{})
		Expect(err).NotTo(HaveOccurred())
		driver = graphDriver.(*aufs.Driver)
		Expect(err).NotTo(HaveOccurred())

		id = "my_banana_id"
		err = driver.Create(id, "")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(driver.Cleanup()).To(Succeed())

		ps, err := gexec.Start(
			exec.Command("umount", graphRoot), GinkgoWriter, GinkgoWriter,
		)
		Expect(err).NotTo(HaveOccurred())
		Eventually(ps).Should(gexec.Exit(0))

		Expect(os.RemoveAll(graphRoot)).To(Succeed())
	})

	It("can get a quotaed layer", func() {
		_, err := driver.GetQuotaed(id, "my_label")
		Expect(err).NotTo(HaveOccurred())
	})

	It("gives me a layer I can write to", func() {
		path, err := driver.GetQuotaed(id, "my_other_label")
		ps, err := gexec.Start(
			exec.Command("touch", fmt.Sprintf("%s/my_test_file", path)),
			GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		Eventually(ps).Should(gexec.Exit(0))

		ps, err = gexec.Start(
			exec.Command("ls", path),
			GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		Eventually(ps).Should(gbytes.Say("my_test_file"))
	})
})
