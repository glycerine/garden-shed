package rootfs_provider_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/garden-shed/layercake"
	"github.com/cloudfoundry-incubator/garden-shed/layercake/fake_cake"
	"github.com/cloudfoundry-incubator/garden-shed/repository_fetcher"
	. "github.com/cloudfoundry-incubator/garden-shed/rootfs_provider"
	"github.com/cloudfoundry-incubator/garden-shed/rootfs_provider/fake_namespacer"
	"github.com/docker/docker/image"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type FakeVolumeCreator struct {
	Created     []RootAndVolume
	CreateError error
}

type RootAndVolume struct {
	RootPath string
	Volume   string
}

func (f *FakeVolumeCreator) Create(path, v string) error {
	f.Created = append(f.Created, RootAndVolume{path, v})
	return f.CreateError
}

var _ = FDescribe("Layer Creator", func() {
	var (
		fakeCake          *fake_cake.FakeCake
		fakeNamespacer    *fake_namespacer.FakeNamespacer
		fakeVolumeCreator *FakeVolumeCreator
		name              string

		provider *ContainerLayerCreator
	)

	BeforeEach(func() {
		fakeCake = new(fake_cake.FakeCake)
		fakeVolumeCreator = &FakeVolumeCreator{}
		fakeNamespacer = &fake_namespacer.FakeNamespacer{}
		name = "some-name"

		provider = NewLayerCreator(
			fakeCake,
			fakeVolumeCreator,
			fakeNamespacer,
		)
	})

	Describe("ProvideRootFS", func() {
		Context("when the namespace parameter is false", func() {
			It("creates a graph entry with it as the parent", func() {
				fakeCake.MountReturns("/some/graph/driver/mount/point", nil)

				mountpoint, envvars, err := provider.Create(
					"some-id",
					&repository_fetcher.Image{
						ImageID: "some-image-id",
						Env:     []string{"env1=env1value", "env2=env2value"},
					},
					false,
					0,
				)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeCake.CreateCallCount()).To(Equal(1))
				id, parent := fakeCake.CreateArgsForCall(0)
				Expect(id).To(Equal(layercake.ContainerID("some-id")))
				Expect(parent).To(Equal(layercake.DockerImageID("some-image-id")))

				Expect(mountpoint).To(Equal("/some/graph/driver/mount/point"))
				Expect(envvars).To(Equal(
					[]string{
						"env1=env1value",
						"env2=env2value",
					},
				))
			})
		})

		Context("when the namespace parameter is true", func() {
			Context("and the image has not been translated yet", func() {
				BeforeEach(func() {
					fakeCake.GetReturns(nil, errors.New("no image here"))

					fakeCake.MountStub = func(id layercake.ID) (string, error) {
						return "/mount/point/" + id.GraphID(), nil
					}

					fakeCake.MountNamespacedStub = func(id layercake.ID) (string, error) {
						return "/mount/point/namespaced/" + id.GraphID(), nil
					}

					fakeNamespacer.CacheKeyReturns("jam")
				})

				It("returns the mount points and environment variables", func() {
					mountpoint, envvars, err := provider.Create(
						"some-id",
						&repository_fetcher.Image{
							ImageID: "some-image-id",
							Env:     []string{"env1=env1value", "env2=env2value"},
						},
						true,
						0,
					)
					Expect(err).ToNot(HaveOccurred())

					Expect(mountpoint).To(Equal("/mount/point/" + layercake.ContainerID("some-id").GraphID()))
					Expect(envvars).To(Equal(
						[]string{
							"env1=env1value",
							"env2=env2value",
						},
					))
				})

				It("creates a graph entry with it as the parent", func() {
					_, _, err := provider.Create(
						"some-id",
						&repository_fetcher.Image{
							ImageID: "some-image-id",
						},
						true,
						0,
					)
					Expect(err).ToNot(HaveOccurred())

					Expect(fakeCake.CreateCallCount()).To(Equal(2))
					id, parent := fakeCake.CreateArgsForCall(0)
					Expect(id).To(Equal(layercake.NamespacedID(layercake.DockerImageID("some-image-id"), "jam")))
					Expect(parent).To(Equal(layercake.DockerImageID("some-image-id")))

					id, parent = fakeCake.CreateArgsForCall(1)
					Expect(id).To(Equal(layercake.ContainerID("some-id")))
					Expect(parent).To(Equal(layercake.NamespacedID(layercake.DockerImageID("some-image-id"), "jam")))
				})

				It("mounts the layers", func() {
					_, _, err := provider.Create(
						"some-id",
						&repository_fetcher.Image{
							ImageID: "some-image-id",
						},
						true,
						0,
					)
					Expect(err).ToNot(HaveOccurred())

					Expect(fakeCake.MountNamespacedCallCount()).To(Equal(1))
					Expect(fakeCake.MountNamespacedArgsForCall(0)).To(Equal(
						layercake.NamespacedID(layercake.DockerImageID("some-image-id"), "jam")))
					Expect(fakeCake.MountCallCount()).To(Equal(1))
					Expect(fakeCake.MountArgsForCall(0)).To(Equal(layercake.ContainerID("some-id")))
				})

				It("namespaces the last layer", func() {
					_, _, err := provider.Create(
						"some-id",
						&repository_fetcher.Image{
							ImageID: "some-image-id",
						},
						true,
						0,
					)
					Expect(err).ToNot(HaveOccurred())

					Expect(fakeNamespacer.NamespaceCallCount()).To(Equal(1))
					dst := fakeNamespacer.NamespaceArgsForCall(0)
					Expect(dst).To(Equal("/mount/point/namespaced/" + layercake.NamespacedID(layercake.DockerImageID("some-image-id"), "jam").GraphID()))
				})
			})

			Context("and the image has already been translated", func() {
				BeforeEach(func() {
					fakeCake.MountStub = func(id layercake.ID) (string, error) {
						return "/mount/point/" + id.GraphID(), nil
					}

					fakeNamespacer.CacheKeyReturns("sandwich")

					fakeCake.GetStub = func(id layercake.ID) (*image.Image, error) {
						if id == (layercake.NamespacedID(layercake.DockerImageID("some-image-id"), "sandwich")) {
							return &image.Image{}, nil
						}

						return nil, errors.New("hello")
					}

				})

				It("reuses the translated layer", func() {
					mountpoint, envvars, err := provider.Create(
						"some-id",
						&repository_fetcher.Image{
							ImageID: "some-image-id",
							Env:     []string{"env1=env1value", "env2=env2value"},
						},
						true,
						0,
					)
					Expect(err).ToNot(HaveOccurred())

					Expect(fakeCake.CreateCallCount()).To(Equal(1))
					id, parent := fakeCake.CreateArgsForCall(0)
					Expect(id).To(Equal(layercake.ContainerID("some-id")))
					Expect(parent).To(Equal(layercake.NamespacedID(layercake.DockerImageID("some-image-id"), "sandwich")))

					Expect(fakeNamespacer.NamespaceCallCount()).To(Equal(0))

					Expect(mountpoint).To(Equal("/mount/point/" + layercake.ContainerID("some-id").GraphID()))
					Expect(envvars).To(Equal(
						[]string{
							"env1=env1value",
							"env2=env2value",
						},
					))
				})
			})
		})

		Context("when the image has associated VOLUMEs", func() {
			It("creates empty directories for all volumes", func() {
				fakeCake.MountReturns("/some/graph/driver/mount/point", nil)

				_, _, err := provider.Create(
					"some-id",
					&repository_fetcher.Image{ImageID: "some-image-id", Volumes: []string{"/foo", "/bar"}},
					false,
					0,
				)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeVolumeCreator.Created).To(Equal(
					[]RootAndVolume{
						{"/some/graph/driver/mount/point", "/foo"},
						{"/some/graph/driver/mount/point", "/bar"},
					}))
			})

			Context("when creating a volume fails", func() {
				It("returns an error", func() {
					fakeCake.MountReturns("/some/graph/driver/mount/point", nil)
					fakeVolumeCreator.CreateError = errors.New("o nooo")

					_, _, err := provider.Create(
						"some-id",
						&repository_fetcher.Image{ImageID: "some-image-id", Volumes: []string{"/foo", "/bar"}},
						false,
						0,
					)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("but creating the graph entry fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeCake.CreateReturns(disaster)
			})

			It("returns the error", func() {
				_, _, err := provider.Create(
					"some-id",
					&repository_fetcher.Image{ImageID: "some-image-id"},
					false,
					0,
				)
				Expect(err).To(Equal(disaster))
			})
		})

		Context("but getting the graph entry fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeCake.MountReturns("", disaster)
			})

			It("returns the error", func() {
				_, _, err := provider.Create(
					"some-id",
					&repository_fetcher.Image{ImageID: "some-image-id"},
					false,
					0,
				)
				Expect(err).To(Equal(disaster))
			})
		})
	})
})
