package aufs_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/cloudfoundry-incubator/garden-shed/docker_drivers/aufs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("LoopLinux", func() {
	var (
		bsFilePath string
		destPath   string
		loop       *aufs.Loop
	)

	BeforeEach(func() {
		var err error

		ensureLoopDevices()

		tempFile, err := ioutil.TempFile("", "")
		Expect(err).NotTo(HaveOccurred())
		bsFilePath = tempFile.Name()
		_, err = exec.Command("truncate", "-s", "10M", bsFilePath).CombinedOutput()
		Expect(err).NotTo(HaveOccurred())
		_, err = exec.Command("mkfs.ext4", "-F", bsFilePath).CombinedOutput()
		Expect(err).NotTo(HaveOccurred())

		destPath, err = ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())

		loop = &aufs.Loop{}
	})

	AfterEach(func() {
		syscall.Unmount(destPath, 0)
		Expect(os.RemoveAll(destPath)).To(Succeed())
		Expect(os.Remove(bsFilePath)).To(Succeed())
	})

	Describe("MountFile", func() {
		It("mounts the file", func() {
			Expect(loop.MountFile(bsFilePath, destPath)).To(Succeed())

			session, err := gexec.Start(exec.Command("mount"), GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gbytes.Say(
				fmt.Sprintf("%s on %s type ext4 \\(rw\\)", bsFilePath, destPath),
			))
		})

		Context("when using a file that does not exist", func() {
			It("should return an error", func() {
				Expect(loop.MountFile("/path/to/my/nonexisting/banana", "/path/to/dest")).To(HaveOccurred())
			})
		})
	})

	Describe("Unmount", func() {
		It("should not leak devices", func() {
			var devicesAfterCreate, devicesAfterRelease int

			destPaths := make([]string, 10)
			for i := 0; i < 10; i++ {
				var err error

				tempFile, err := ioutil.TempFile("", "")
				Expect(err).NotTo(HaveOccurred())

				_, err = exec.Command("truncate", "-s", "10M", tempFile.Name()).CombinedOutput()
				Expect(err).NotTo(HaveOccurred())
				_, err = exec.Command("mkfs.ext4", "-F", tempFile.Name()).CombinedOutput()
				Expect(err).NotTo(HaveOccurred())

				destPaths[i], err = ioutil.TempDir("", "")
				Expect(err).NotTo(HaveOccurred())

				Expect(loop.MountFile(tempFile.Name(), destPaths[i])).To(Succeed())

				// Expect(os.Remove(tempFile.Name())).To(Succeed())
			}

			output, err := exec.Command("sh", "-c", "losetup -a | wc -l").CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			devicesAfterCreate, err = strconv.Atoi(strings.TrimSpace(string(output)))
			Expect(err).NotTo(HaveOccurred())

			for i := 0; i < 10; i++ {
				Expect(loop.Unmount(destPaths[i])).To(Succeed())
			}

			output, err = exec.Command("sh", "-c", "losetup -a | wc -l").CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			devicesAfterRelease, err = strconv.Atoi(strings.TrimSpace(string(output)))
			Expect(err).NotTo(HaveOccurred())

			Expect(devicesAfterRelease).To(BeNumerically("~", devicesAfterCreate-10, 2))
		})

		Context("when the provided mount point does not exist", func() {
			It("should return an error", func() {
				Expect(loop.Unmount("/dev/loopbanana")).NotTo(Succeed())
			})
		})
	})
})

func ensureLoopDevices() {
	permitDeviceAccess()
	for i := 0; i < 128; i++ {
		// ignoring errors if it exists..
		exec.Command("mknod", "-m", "0660", fmt.Sprintf("/dev/loop%d", i), "b", "7", fmt.Sprintf("%d", i)).Run()
		Expect(fmt.Sprintf("/dev/loop%d", i)).To(BeAnExistingFile(), "loop device should exist or have been created")
	}
}

func permitDeviceAccess() {
	out, err := exec.Command("sh", "-c", `
		devices_mount_info=$(cat /proc/self/cgroup | grep devices)
		if [ -z "$devices_mount_info" ]; then
			# cgroups not set up; must not be in a container
			exit 0
		fi
		devices_subsytems=$(echo $devices_mount_info | cut -d: -f2)
		devices_subdir=$(echo $devices_mount_info | cut -d: -f3)
		if [ "$devices_subdir" = "/" ]; then
			# we're in the root devices cgroup; must not be in a container
			exit 0
		fi
		cgroup_dir=/tmp/garden-devices-cgroup
		if [ ! -e ${cgroup_dir} ]; then
			# mount our container's devices subsystem somewhere
			mkdir ${cgroup_dir}
		fi
		if ! mountpoint -q ${cgroup_dir}; then
			mount -t cgroup -o $devices_subsytems none ${cgroup_dir}
		fi
		# permit our cgroup to do everything with all devices
		echo a > ${cgroup_dir}${devices_subdir}/devices.allow
	`).CombinedOutput()

	Expect(err).NotTo(HaveOccurred(), string(out))
}
