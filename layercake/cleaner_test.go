package layercake_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/garden-shed/layercake"
	"github.com/cloudfoundry-incubator/garden-shed/layercake/fake_cake"
	"github.com/docker/docker/image"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Oven cleaner", func() {
	var (
		retainer           layercake.RetainChecker
		gc                 *layercake.OvenCleaner
		fakeCake           *fake_cake.FakeCake
		child2parent       map[layercake.ID]layercake.ID // child -> parent
		enableImageCleanup bool
	)

	BeforeEach(func() {
		enableImageCleanup = true

		retainer = layercake.NewRetainer()

		fakeCake = new(fake_cake.FakeCake)
		fakeCake.GetStub = func(id layercake.ID) (*image.Image, error) {
			if parent, ok := child2parent[id]; ok {
				return &image.Image{ID: id.GraphID(), Parent: parent.GraphID()}, nil
			}

			return &image.Image{}, nil
		}

		fakeCake.IsLeafStub = func(id layercake.ID) (bool, error) {
			for _, p := range child2parent {
				if p == id {
					return false, nil
				}
			}

			return true, nil
		}

		fakeCake.RemoveStub = func(id layercake.ID) error {
			delete(child2parent, id)
			return nil
		}

		child2parent = make(map[layercake.ID]layercake.ID)
	})

	JustBeforeEach(func() {
		gc = layercake.NewOvenCleaner(
			fakeCake,
			lagertest.NewTestLogger("test"),
			enableImageCleanup,
			retainer,
		)
	})

	Describe("Remove", func() {
		Context("when the layer has no parents", func() {
			BeforeEach(func() {
				fakeCake.GetReturns(&image.Image{}, nil)
			})

			It("removes the layer", func() {
				Expect(gc.Remove(layercake.ContainerID("child"))).To(Succeed())
				Expect(fakeCake.RemoveCallCount()).To(Equal(1))
				Expect(fakeCake.RemoveArgsForCall(0)).To(Equal(layercake.ContainerID("child")))
			})

			Context("when the layer is retained", func() {
				JustBeforeEach(func() {
					retainer.Retain(lagertest.NewTestLogger(""), layercake.ContainerID("child"))
				})

				It("should not remove the layer", func() {
					Expect(gc.Remove(layercake.ContainerID("child"))).To(Succeed())
					Expect(fakeCake.RemoveCallCount()).To(Equal(0))
				})
			})
		})

		Context("when removing fails", func() {
			It("returns an error", func() {
				fakeCake.RemoveReturns(errors.New("cake failure"))
				Expect(gc.Remove(layercake.ContainerID("whatever"))).To(MatchError("cake failure"))
			})
		})

		Context("when the layer has a parent", func() {
			BeforeEach(func() {
				child2parent[layercake.ContainerID("child")] = layercake.DockerImageID("parent")
			})

			Context("and the parent has no other children", func() {
				It("removes the layer, and its parent", func() {
					Expect(gc.Remove(layercake.ContainerID("child"))).To(Succeed())

					Expect(fakeCake.RemoveCallCount()).To(Equal(2))
					Expect(fakeCake.RemoveArgsForCall(0)).To(Equal(layercake.ContainerID("child")))
					Expect(fakeCake.RemoveArgsForCall(1)).To(Equal(layercake.DockerImageID("parent")))
				})
			})

			Context("when removing fails", func() {
				It("does not remove any more layers", func() {
					fakeCake.RemoveReturns(errors.New("cake failure"))
					gc.Remove(layercake.ContainerID("whatever"))
					Expect(fakeCake.RemoveCallCount()).To(Equal(1))
				})
			})

			Context("but the layer has another child", func() {
				BeforeEach(func() {
					child2parent[layercake.ContainerID("some-other-child")] = layercake.DockerImageID("parent")
				})

				It("removes only the initial layer", func() {
					child2parent[layercake.ContainerID("child")] = layercake.DockerImageID("parent")
					Expect(gc.Remove(layercake.ContainerID("child"))).To(Succeed())

					Expect(fakeCake.RemoveCallCount()).To(Equal(1))
					Expect(fakeCake.RemoveArgsForCall(0)).To(Equal(layercake.ContainerID("child")))
				})
			})
		})

		Context("when the layer has grandparents", func() {
			It("removes all the grandparents", func() {
				child2parent[layercake.ContainerID("child")] = layercake.DockerImageID("parent")
				child2parent[layercake.DockerImageID("parent")] = layercake.DockerImageID("granddaddy")

				Expect(gc.Remove(layercake.ContainerID("child"))).To(Succeed())

				Expect(fakeCake.RemoveCallCount()).To(Equal(3))
				Expect(fakeCake.RemoveArgsForCall(0)).To(Equal(layercake.ContainerID("child")))
				Expect(fakeCake.RemoveArgsForCall(1)).To(Equal(layercake.DockerImageID("parent")))
				Expect(fakeCake.RemoveArgsForCall(2)).To(Equal(layercake.DockerImageID("granddaddy")))
			})
		})

		Context("when image cleanup is disabled", func() {
			BeforeEach(func() {
				enableImageCleanup = false

				child2parent[layercake.ContainerID("child")] = layercake.DockerImageID("parent")
			})

			It("removes the container layer but not the parent layer", func() {
				Expect(gc.Remove(layercake.ContainerID("child"))).To(Succeed())

				Expect(fakeCake.RemoveCallCount()).To(Equal(1))
				Expect(fakeCake.RemoveArgsForCall(0)).To(Equal(layercake.ContainerID("child")))
			})
		})
	})
})
