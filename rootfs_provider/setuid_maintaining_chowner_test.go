package rootfs_provider

import (
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = FDescribe("chowning", func() {
	var chownFunc = func(path string, uid, gid int) error {
		return nil
	}

	It("maintains the setuid bit", func() {
		myFile, err := ioutil.TempFile("", "")
		Expect(err).NotTo(HaveOccurred())
		defer func() { Expect(os.Remove(myFile.Name())).To(Succeed()) }()

		sess, err := gexec.Start(exec.Command("chmod", "u+s", myFile.Name()), GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess).Should(gexec.Exit(0))

		info, err := os.Stat(myFile.Name())
		Expect(err).ToNot(HaveOccurred())
		Expect(info.Mode() & os.ModeSetuid).ToNot(Equal(os.FileMode(0)))

		chownFunc(myFile.Name(), 100, 100)

		info, err = os.Stat(myFile.Name())
		Expect(err).ToNot(HaveOccurred())
		Expect(info.Mode() & os.ModeSetuid).ToNot(Equal(os.FileMode(0)))
	})

	It("changes the gid and uid", func() {
		myFile, err := ioutil.TempFile("", "")
		Expect(err).NotTo(HaveOccurred())
		defer func() { Expect(os.Remove(myFile.Name())).To(Succeed()) }()

		info, err := os.Stat(myFile.Name())
		Expect(info.Sys().(*syscall.Stat_t).Uid).ToNot(Equal(100))
		Expect(info.Sys().(*syscall.Stat_t).Gid).ToNot(Equal(100))

		chownFunc(myFile.Name(), 100, 100)

		info, err = os.Stat(myFile.Name())
		Expect(info.Sys().(*syscall.Stat_t).Uid).To(Equal(100))
		Expect(info.Sys().(*syscall.Stat_t).Gid).To(Equal(100))
	})
})
