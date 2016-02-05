package rootfs_provider_test

import (
	"errors"
	"net/url"

	"github.com/cloudfoundry-incubator/garden-shed/layercake"
	"github.com/cloudfoundry-incubator/garden-shed/layercake/fake_cake"
	"github.com/cloudfoundry-incubator/garden-shed/layercake/fake_retainer"
	"github.com/cloudfoundry-incubator/garden-shed/repository_fetcher"
	"github.com/cloudfoundry-incubator/garden-shed/rootfs_provider"
	"github.com/cloudfoundry-incubator/garden-shed/rootfs_provider/fakes"
	"github.com/hashicorp/go-multierror"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("The Cake Co-ordinator", func() {
	var (
		fakeFetcher      *fakes.FakeRepositoryFetcher
		fakeLayerCreator *fakes.FakeLayerCreator
		fakeCake         *fake_cake.FakeCake
		fakeRetainer     *fake_retainer.FakeRetainer
		logger           *lagertest.TestLogger

		cakeOrdinator *rootfs_provider.CakeOrdinator
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")

		fakeFetcher = new(fakes.FakeRepositoryFetcher)

		fakeRetainer = new(fake_retainer.FakeRetainer)
		fakeLayerCreator = new(fakes.FakeLayerCreator)
		fakeCake = new(fake_cake.FakeCake)
		cakeOrdinator = rootfs_provider.NewCakeOrdinator(fakeCake, fakeFetcher, fakeLayerCreator, fakeRetainer)
	})

	Describe("creating container layers", func() {
		Context("When the image is succesfully fetched", func() {
			It("creates a container layer on top of the fetched layer", func() {
				image := &repository_fetcher.Image{ImageID: "my cool image"}
				fakeFetcher.FetchReturns(image, nil)
				fakeLayerCreator.CreateReturns("potato", []string{"foo=bar"}, errors.New("cake"))

				spec := rootfs_provider.Spec{
					RootFS:     &url.URL{Path: "parent"},
					Namespaced: true,
					QuotaSize:  55,
				}
				rootfsPath, envs, err := cakeOrdinator.Create(logger, "container-id", spec)
				Expect(rootfsPath).To(Equal("potato"))
				Expect(envs).To(Equal([]string{"foo=bar"}))
				Expect(err).To(MatchError("cake"))

				Expect(fakeLayerCreator.CreateCallCount()).To(Equal(1))
				containerID, parentImage, layerCreatorSpec := fakeLayerCreator.CreateArgsForCall(0)
				Expect(containerID).To(Equal("container-id"))
				Expect(parentImage).To(Equal(image))
				Expect(layerCreatorSpec).To(Equal(spec))
			})
		})

		Context("when fetching fails", func() {
			It("returns an error", func() {
				fakeFetcher.FetchReturns(nil, errors.New("amadeus"))
				_, _, err := cakeOrdinator.Create(logger, "", rootfs_provider.Spec{
					RootFS:     nil,
					Namespaced: true,
					QuotaSize:  12,
				})
				Expect(err).To(MatchError("amadeus"))
			})
		})

		Context("when the quota scope is exclusive", func() {
			It("disables quota for the fetcher", func() {
				_, _, err := cakeOrdinator.Create(logger, "", rootfs_provider.Spec{
					RootFS:     &url.URL{},
					Namespaced: false,
					QuotaSize:  33,
					QuotaScope: rootfs_provider.QuotaScopeExclusive,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeFetcher.FetchCallCount()).To(Equal(1))
				_, diskQuota := fakeFetcher.FetchArgsForCall(0)
				Expect(diskQuota).To(BeNumerically("==", 0))
			})
		})

		Context("when the quota scope is total", func() {
			It("passes down the same quota number to the fetcher", func() {
				_, _, err := cakeOrdinator.Create(logger, "", rootfs_provider.Spec{
					RootFS:     &url.URL{},
					Namespaced: false,
					QuotaSize:  33,
					QuotaScope: rootfs_provider.QuotaScopeTotal,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeFetcher.FetchCallCount()).To(Equal(1))
				_, diskQuota := fakeFetcher.FetchArgsForCall(0)
				Expect(diskQuota).To(BeNumerically("==", 33))
			})
		})
	})

	Describe("Retain", func() {
		It("can be retained by Retainer", func() {
			retainedId := layercake.ContainerID("banana")
			cakeOrdinator.Retain(logger, retainedId)

			Expect(fakeRetainer.RetainCallCount()).To(Equal(1))
			_, id := fakeRetainer.RetainArgsForCall(0)
			Expect(id).To(Equal(retainedId))
		})
	})

	Describe("Destroy", func() {
		It("delegates removals", func() {
			fakeCake.GetAllLeavesReturns([]string{"1", "2", "3"}, nil)

			err := cakeOrdinator.Destroy(logger, "something")
			Expect(fakeCake.RemoveCallCount()).To(Equal(3))

			id := fakeCake.RemoveArgsForCall(0)
			Expect(id).To(Equal(layercake.DockerImageID("1")))
			id = fakeCake.RemoveArgsForCall(1)
			Expect(id).To(Equal(layercake.DockerImageID("2")))
			id = fakeCake.RemoveArgsForCall(2)
			Expect(id).To(Equal(layercake.DockerImageID("3")))

			Expect(err).NotTo(HaveOccurred())
		})

		Context("when retrieving all leaves fails", func() {
			It("returns the error", func() {
				fakeCake.GetAllLeavesReturns([]string{}, errors.New("some-error"))
				err := cakeOrdinator.Destroy(logger, "something")

				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("when cleanup fails for a single leaf", func() {
			It("returns the error", func() {
				fakeCake.GetAllLeavesReturns([]string{"yo"}, nil)
				fakeCake.RemoveReturns(errors.New("single-error"))

				err := cakeOrdinator.Destroy(logger, "whatever")
				multiErr := err.(*multierror.Error)
				Expect(multiErr.Errors[0]).To(MatchError("single-error"))
			})
		})

		Context("when cleanup fails for multiple leaves", func() {
			var err error

			BeforeEach(func() {
				fakeCake.GetAllLeavesReturns([]string{"first", "second", "third"}, nil)

				fakeCake.RemoveStub = func(id layercake.ID) error {
					if id == layercake.DockerImageID("first") {
						return errors.New("error-first")
					} else if id == layercake.DockerImageID("second") {
						return errors.New("error-second")
					}
					return nil
				}

				err = cakeOrdinator.Destroy(logger, "whatever")
			})

			It("returns all errors", func() {
				multiErr := err.(*multierror.Error)

				Expect(multiErr.Errors).To(HaveLen(2))
				Expect(multiErr.Errors[0]).To(MatchError("error-first"))
				Expect(multiErr.Errors[1]).To(MatchError("error-second"))
			})

			It("calls Destroy for all leaves in the graph", func() {
				Expect(fakeCake.RemoveCallCount()).To(Equal(3))
			})
		})

		It("prevents concurrent garbage collection and creation", func() {
			fakeCake.GetAllLeavesReturns([]string{"1"}, nil)

			removeStarted := make(chan struct{})
			removeReturns := make(chan struct{})
			fakeCake.RemoveStub = func(id layercake.ID) error {
				close(removeStarted)
				<-removeReturns
				return nil
			}

			go cakeOrdinator.Destroy(logger, "")
			<-removeStarted
			go cakeOrdinator.Create(logger, "", rootfs_provider.Spec{
				RootFS:     &url.URL{},
				Namespaced: false,
				QuotaSize:  33,
			})

			Consistently(fakeFetcher.FetchCallCount).Should(Equal(0))
			close(removeReturns)
			Eventually(fakeFetcher.FetchCallCount).Should(Equal(1))
		})
	})

	It("allows concurrent creation as long as deletion is not ongoing", func() {
		fakeBlocks := make(chan struct{})
		fakeFetcher.FetchStub = func(*url.URL, int64) (*repository_fetcher.Image, error) {
			<-fakeBlocks
			return nil, nil
		}

		go cakeOrdinator.Create(logger, "", rootfs_provider.Spec{
			RootFS:     &url.URL{},
			Namespaced: false,
			QuotaSize:  33,
		})
		go cakeOrdinator.Create(logger, "", rootfs_provider.Spec{
			RootFS:     &url.URL{},
			Namespaced: false,
			QuotaSize:  33,
		})

		Eventually(fakeFetcher.FetchCallCount).Should(Equal(2))
		close(fakeBlocks)
	})
})
