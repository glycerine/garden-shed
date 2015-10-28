package aufs_test

import (
	"errors"
	"path/filepath"

	"github.com/cloudfoundry-incubator/garden-shed/docker_drivers/aufs"
	"github.com/cloudfoundry-incubator/garden-shed/docker_drivers/aufs/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Aufs", func() {
	var (
		fakeGraphDriver        *fakes.FakeGraphDriver
		fakeQuotaLayerProvider *fakes.FakeQuotaLayerProvider

		driver *aufs.Driver

		rootPath string
	)

	BeforeEach(func() {
		fakeGraphDriver = new(fakes.FakeGraphDriver)
		fakeQuotaLayerProvider = new(fakes.FakeQuotaLayerProvider)

		rootPath = "/path/to/my/banana/graph"
		driver = &aufs.Driver{fakeGraphDriver, fakeQuotaLayerProvider, rootPath}
	})

	Describe("GetQuotaed", func() {
		It("should call the quota layer provider", func() {
			id := "banana-id"
			quota := int64(10 * 1024 * 1024)

			_, err := driver.GetQuotaed(id, "", quota)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeQuotaLayerProvider.ProvideCallCount()).To(Equal(1))
			destinationPath, actualQuota := fakeQuotaLayerProvider.ProvideArgsForCall(0)
			Expect(actualQuota).To(Equal(quota))
			Expect(destinationPath).To(Equal(filepath.Join(rootPath, "aufs", "diff", id)))
		})

		Context("when the quota layer provider fails", func() {
			BeforeEach(func() {
				fakeQuotaLayerProvider.ProvideReturns(errors.New("My banana went bad"))
			})

			It("should return an error", func() {
				_, err := driver.GetQuotaed("banana-id", "", 10*1024*1024)
				Expect(err).To(MatchError(ContainSubstring("My banana went bad")))
			})

			It("should not mount the layer", func() {
				driver.GetQuotaed("banana-id", "", 10*1024*1024)
				Expect(fakeGraphDriver.GetCallCount()).To(Equal(0))
			})
		})

		It("should call the GraphDriver's Get method", func() {
			id := "mango-id"
			mountLabel := "wild mangos: handle with care"
			quota := int64(12 * 1024 * 1024)

			_, err := driver.GetQuotaed(id, mountLabel, quota)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeGraphDriver.GetCallCount()).To(Equal(1))
			gottenID, gottenMountLabel := fakeGraphDriver.GetArgsForCall(0)
			Expect(gottenID).To(Equal(id))
			Expect(gottenMountLabel).To(Equal(mountLabel))
		})

		It("should return the path gotten from the GraphDriver", func() {
			mountPath := "/path/to/mounted/banana"

			fakeGraphDriver.GetReturns(mountPath, nil)

			path, err := driver.GetQuotaed("test-banana-id", "", 10*1024*1024)
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(mountPath))
		})

		Context("when GraphDriver fails to mount the layer", func() {
			It("should return an error", func() {
				fakeGraphDriver.GetReturns("", errors.New("Another banana error"))

				_, err := driver.GetQuotaed("banana-id", "", 10*1024*1024)
				Expect(err).To(MatchError(ContainSubstring("Another banana error")))
			})
		})
	})

	Describe("RemoveQuotaed", func() {
		It("should call the GraphDriver's Remove method", func() {
			id := "herring-id"

			Expect(driver.RemoveQuotaed(id)).To(Succeed())
			Expect(fakeGraphDriver.RemoveCallCount()).To(Equal(1))
			Expect(fakeGraphDriver.RemoveArgsForCall(0)).To(Equal(id))
		})

		Context("when the GraphDriver fails to remove the layer", func() {
			It("should return an error", func() {
				fakeGraphDriver.RemoveReturns(errors.New("herring smell"))

				Expect(driver.RemoveQuotaed("an-id")).To(MatchError(ContainSubstring("herring smell")))
			})
		})

		It("should call the quota layer provider", func() {
			id := "trout-id"

			Expect(driver.RemoveQuotaed(id)).To(Succeed())
			Expect(fakeQuotaLayerProvider.DestroyCallCount()).To(Equal(1))
			Expect(fakeQuotaLayerProvider.DestroyArgsForCall(0)).To(Equal(filepath.Join(rootPath, "aufs", "diff", id)))
		})

		Context("when the quota layer provider fails to destroy the layer", func() {
			It("should return an error", func() {
				fakeQuotaLayerProvider.DestroyReturns(errors.New("rotten trout"))

				Expect(driver.RemoveQuotaed("an-id")).To(MatchError(ContainSubstring("rotten trout")))
			})
		})
	})
})
