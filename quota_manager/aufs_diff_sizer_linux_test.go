package quota_manager_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/cloudfoundry-incubator/garden-linux/linux_container/quota_manager"
	"github.com/cloudfoundry-incubator/garden-shed/quota_manager/fakes"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("AufsLayerSizer", func() {
	var (
		diffSizer              *quota_manager.AUFSDiffSizer
		fakeAUFSDiffPathFinder *fakes.FakeAUFSDiffPathFinder
		mountDir               string
		backingFile            *os.File
		logger                 lager.Logger
	)

	Describe("DiffSize", func() {
		const quotaMB = 10

		BeforeEach(func() {
			fakeAUFSDiffPathFinder = new(fakes.FakeAUFSDiffPathFinder)

			diffSizer = &quota_manager.AUFSDiffSizer{
				fakeAUFSDiffPathFinder,
			}

			logger = lagertest.NewTestLogger("AUFSQuotaManager-test")

			var err error
			mountDir, err = ioutil.TempDir("", "quota_manager_test")
			Expect(err).NotTo(HaveOccurred())

			backingFile, err = ioutil.TempFile("", "quota_manager_backing_store")
			Expect(err).NotTo(HaveOccurred())

			session, err := gexec.Start(exec.Command("truncate", "-s", fmt.Sprintf("%dM", quotaMB), backingFile.Name()), GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			session, err = gexec.Start(exec.Command("mkfs.ext4", "-F", backingFile.Name()), GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			session, err = gexec.Start(exec.Command("mount", "-o", "loop", backingFile.Name(), mountDir), GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
		})

		AfterEach(func() {
			session, err := gexec.Start(exec.Command("umount", mountDir), GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			Expect(os.RemoveAll(mountDir)).To(Succeed())
			Expect(os.RemoveAll(backingFile.Name())).To(Succeed())
		})

		Context("when the rootFSPath does not exist", func() {
			It("returns an error", func() {
				_, err := diffSizer.DiffSize(logger, "does not exist")
				Expect(err).To(MatchError(ContainSubstring("does not exist")))
			})
		})

		Context("when the rootFSPath exists", func() {
			It("correctly delegates the call to the AUFSDiffPathFinder", func() {
				_, err := diffSizer.DiffSize(logger, mountDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeAUFSDiffPathFinder.GetDiffLayerPathCallCount()).To(Equal(1))
				Expect(fakeAUFSDiffPathFinder.GetDiffLayerPathArgsForCall(0)).To(Equal(mountDir))
			})

			Context("when the directory from AUFSDiffPathFinder does not match with a mounted filesystem", func() {
				BeforeEach(func() {
					fakeAUFSDiffPathFinder.GetDiffLayerPathReturns("this/path/is/not/spiderman")
				})

				It("returns 0", func() {
					tempDir, err := ioutil.TempDir("", "spiderman")
					Expect(err).NotTo(HaveOccurred())

					stats, err := diffSizer.DiffSize(logger, tempDir)
					Expect(err).NotTo(HaveOccurred())
					Expect(stats).To(Equal(uint64(0)))
					Expect(os.RemoveAll(tempDir)).To(Succeed())
				})
			})

			Context("when the directory from AUFSDiffPathFinder does match the mount-point of an initially empty mounted FS", func() {
				var initialUsage uint64

				BeforeEach(func() {
					fakeAUFSDiffPathFinder.GetDiffLayerPathStub = func(rootFSPath string) string {
						return rootFSPath
					}
					stats, err := diffSizer.DiffSize(logger, mountDir)
					Expect(err).NotTo(HaveOccurred())
					initialUsage = stats
				})

				It("returns stats with only the filesystem metadata", func() {
					Expect(initialUsage).To(BeNumerically("<", quotaMB*1024*1024*0.015)) // metadata of 1.5% of the total quota
				})

				Context("when we write stuff to the FS", func() {
					BeforeEach(func() {
						session, err := gexec.Start(exec.Command("dd", "if=/dev/zero", fmt.Sprintf("of=%s/some-file", mountDir), "bs=1M", "count=7"), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						Eventually(session).Should(gexec.Exit(0))
					})

					It("returns accurate stats", func() {
						stats, err := diffSizer.DiffSize(logger, mountDir)
						Expect(err).NotTo(HaveOccurred())
						Expect(stats).To(Equal(initialUsage + 7*1024*1024))
					})
				})
			})
		})
	})
})
