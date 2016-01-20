package rootfs_provider_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/garden-shed/rootfs_provider"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("BindMounter", func() {
	var (
		mounter    rootfs_provider.BindMounter
		rootFsPath string
		log        lager.Logger
		srcPath    string
		tmpDirPath string
	)

	BeforeEach(func() {
		mounter = rootfs_provider.BindMounter{}

		var err error
		tmpDirPath, err = ioutil.TempDir("", "bind-mounter-test")
		Expect(err).ToNot(HaveOccurred())

		rootFsPath = filepath.Join(tmpDirPath, "rootfs")
		Expect(os.Mkdir(rootFsPath, 755)).ToNot(HaveOccurred())

		srcPath = filepath.Join(tmpDirPath, "src")
		Expect(os.Mkdir(srcPath, 0755)).ToNot(HaveOccurred())

		Expect(ioutil.WriteFile(filepath.Join(srcPath, "file.txt"), []byte{}, 0755)).ToNot(HaveOccurred())

		log = lagertest.NewTestLogger("bind-mounter-test")
	})

	AfterEach(func() {
		syscall.Unmount(srcPath, 0)
		os.RemoveAll(tmpDirPath)
	})

	It("should bind source folder to destination folder", func() {
		err := mounter.Bind(log, rootFsPath, garden.BindMount{
			SrcPath: srcPath,
			DstPath: "foo/bar",
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(filepath.Join(rootFsPath, "foo", "bar", "file.txt")).To(BeARegularFile())
	})

})
