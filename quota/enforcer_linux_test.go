package quota_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	"github.com/cloudfoundry-incubator/garden-shed/quota"
	"github.com/cloudfoundry-incubator/garden-shed/quota/xfs"
	"github.com/docker/docker/daemon/graphdriver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	_ "github.com/docker/docker/daemon/graphdriver/overlay"
)

var _ = Describe("Enforcing a Quota", func() {
	var (
		enforcer     quota.Enforcer
		backingStore string
		mountPoint   string
		volume       string

		logger lager.Logger
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		backingStore = mktmp()
		mountPoint = mktmpdir()

		ensureLoopDevices()

		Expect(xfs.New(logger, backingStore, mountPoint).Create(1024 * 1024 * 50)).To(Succeed())
		volume = path.Join(mountPoint, "volume")
		Expect(os.MkdirAll(volume, 0777)).To(Succeed())

		enforcer = quota.NewXFSEnforcer(mountPoint)
	})

	AfterEach(func() {
		Expect(xfs.New(logger, backingStore, mountPoint).Delete()).To(Succeed())
		Expect(os.RemoveAll(mountPoint)).To(Succeed())
		Expect(os.RemoveAll(backingStore)).To(Succeed())
	})

	Context("when the write is over the enforced limit", func() {
		It("enforces the quota", func() {
			buff := make([]byte, 100000)
			Expect(enforcer.Enforce(logger, volume, 1000)).To(Succeed())
			Expect(ioutil.WriteFile(path.Join(volume, "big"), buff, 0700)).
				To(MatchError(ContainSubstring("disk quota exceeded")))
		})
	})

	Context("with an overlay directory", func() {
		var layer string

		BeforeEach(func() {
			driver, err := graphdriver.GetDriver("overlay", mountPoint, nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(driver.Create("base", "")).To(Succeed())
			base, err := driver.Get("base", "")
			Expect(err).NotTo(HaveOccurred())

			baseFileData := make([]byte, 100000)
			ioutil.WriteFile(path.Join(base, "some-file"), baseFileData, 0700)

			Expect(driver.Create("layer", "")).To(Succeed())
			layer, err = driver.Get("layer", "")
			Expect(err).NotTo(HaveOccurred())
		})

		FIt("can enforce an exclusive quota", func() {
			Expect(enforcer.Enforce(logger, layer, 1000)).To(Succeed())

			layerData := make([]byte, 100000)
			Expect(ioutil.WriteFile(path.Join(layer, "another-file"), layerData, 0700)).NotTo(Succeed())
		})
	})
})

func mktmp() string {
	tmp, err := ioutil.TempFile("", "")
	Expect(err).NotTo(HaveOccurred())
	return tmp.Name()
}

func mktmpdir() string {
	tmp, err := ioutil.TempDir("", "")
	Expect(err).NotTo(HaveOccurred())
	return tmp
}

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
