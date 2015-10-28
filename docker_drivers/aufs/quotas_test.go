package aufs_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/garden-shed/docker_drivers/aufs"
	"github.com/cloudfoundry-incubator/garden-shed/docker_drivers/aufs/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("QuotaLayerProvider", func() {
	var (
		fakeLoopMounter     *fakes.FakeLoopMounter
		fakeBackingStoreMgr *fakes.FakeBackingStoreMgr
		quotaLayerProvider  *aufs.QuotaLayer
	)

	BeforeEach(func() {
		fakeLoopMounter = new(fakes.FakeLoopMounter)
		fakeBackingStoreMgr = new(fakes.FakeBackingStoreMgr)

		quotaLayerProvider = &aufs.QuotaLayer{
			BackingStoreMgr: fakeBackingStoreMgr,
			LoopMounter:     fakeLoopMounter,
		}
	})

	Describe("Provide", func() {
		It("should create a backing store file", func() {
			reqQuota := int64(12 * 1024)
			Expect(quotaLayerProvider.Provide("/path/to/my/banana", reqQuota)).To(Succeed())

			Expect(fakeBackingStoreMgr.CreateCallCount()).To(Equal(1))
			_, quota := fakeBackingStoreMgr.CreateArgsForCall(0)
			Expect(quota).To(Equal(reqQuota))
		})

		Context("when failing to create a backing store", func() {
			It("should return an error", func() {
				fakeBackingStoreMgr.CreateReturns("", errors.New("create failed!"))

				err := quotaLayerProvider.Provide("/path/to/my/banana", 0)
				Expect(err).To(MatchError("create failed!"))
			})
		})

		It("should mount the backing store file", func() {
			realDevicePath := "/path/to/my/banana/device"
			realDestPath := "/path/to/my/banana"

			fakeBackingStoreMgr.CreateReturns(realDevicePath, nil)

			Expect(quotaLayerProvider.Provide(realDestPath, 0)).To(Succeed())

			Expect(fakeLoopMounter.MountFileCallCount()).To(Equal(1))
			devicePath, destPath := fakeLoopMounter.MountFileArgsForCall(0)
			Expect(devicePath).To(Equal(realDevicePath))
			Expect(destPath).To(Equal(realDestPath))
		})

		Context("when failing to mount the backing store", func() {
			It("should return an error", func() {
				fakeLoopMounter.MountFileReturns(errors.New("another banana error"))

				err := quotaLayerProvider.Provide("/some/path", 0)
				Expect(err).To(MatchError("another banana error"))
			})
		})
	})

	Describe("Destroy", func() {
		It("should unmount the dest path", func() {
			dest := "/some/path"
			quotaLayerProvider.Destroy(dest)

			Expect(fakeLoopMounter.UnmountCallCount()).To(Equal(1))
			Expect(fakeLoopMounter.UnmountArgsForCall(0)).To(Equal(dest))
		})

		Context("when failing to unmount the backing store", func() {
			It("should return an error", func() {
				fakeLoopMounter.UnmountReturns(errors.New("another banana error"))

				err := quotaLayerProvider.Destroy("/some/path")
				Expect(err).To(MatchError("another banana error"))
			})
		})

		It("should delete the correct backing store", func() {
			realDestPath := "/path/to/my/banana"
			quotaLayerProvider.Provide(realDestPath, 0)

			createdId, _ := fakeBackingStoreMgr.CreateArgsForCall(0)

			quotaLayerProvider.Destroy(realDestPath)
			Expect(fakeBackingStoreMgr.DeleteCallCount()).To(Equal(1))
			Expect(fakeBackingStoreMgr.DeleteArgsForCall(0)).To(Equal(createdId))
		})

		Context("when failing to delete the backing store", func() {
			It("should return an error", func() {
				fakeBackingStoreMgr.DeleteReturns(errors.New("create failed!"))

				err := quotaLayerProvider.Destroy("/path/to/my/banana")
				Expect(err).To(MatchError("create failed!"))
			})
		})
	})
})
