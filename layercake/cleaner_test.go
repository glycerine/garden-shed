package layercake_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/garden-shed/layercake"
	"github.com/cloudfoundry-incubator/garden-shed/layercake/fake_cake"
	"github.com/docker/docker/image"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = FDescribe("Oven cleaner", func() {
	var (
		retainer              layercake.RetainChecker
		gc                    *layercake.OvenCleaner
		fakeCake              *fake_cake.FakeCake
		child2parent          map[layercake.ID]layercake.ID // child -> parent
		graphCleanupThreshold int
		logger                lager.Logger
	)

	BeforeEach(func() {
		graphCleanupThreshold = 0
		logger = lagertest.NewTestLogger("test")

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
			retainer,
			graphCleanupThreshold,
		)
	})

	Context("when the threshold is exceeded", func() {

		Describe("GC", func() {
			Context("when there is a single leaf", func() {
				BeforeEach(func() {
					fakeCake.GetAllLeavesReturns([]layercake.ID{layercake.DockerImageID("child")}, nil)
				})

				It("should not remove it when it is used by a container", func() {
					fakeCake.GetReturns(&image.Image{Container: "used-by-me"}, nil)
					Expect(gc.GC(logger, fakeCake)).To(Succeed())
					Expect(fakeCake.RemoveCallCount()).To(Equal(0))
				})

				Context("when the layer has no parents", func() {
					BeforeEach(func() {
						fakeCake.GetReturns(&image.Image{}, nil)
					})

					It("removes the layer", func() {
						Expect(gc.GC(logger, fakeCake)).To(Succeed())
						Expect(fakeCake.RemoveCallCount()).To(Equal(1))
						Expect(fakeCake.RemoveArgsForCall(0)).To(Equal(layercake.DockerImageID("child")))
					})

					Context("when the layer is retained", func() {
						JustBeforeEach(func() {
							retainer.Retain(lagertest.NewTestLogger(""), layercake.DockerImageID("child"))
						})

						It("should not remove the layer", func() {
							Expect(gc.GC(logger, fakeCake)).To(Succeed())
							Expect(fakeCake.RemoveCallCount()).To(Equal(0))
						})
					})

					Context("when removing fails", func() {
						It("returns an error", func() {
							fakeCake.RemoveReturns(errors.New("cake failure"))
							Expect(gc.GC(logger, fakeCake)).To(MatchError("cake failure"))
						})
					})
				})

				Context("when the layer has a parent", func() {
					BeforeEach(func() {
						child2parent[layercake.DockerImageID("child")] = layercake.DockerImageID("parent")
					})

					Context("and the parent has no other children", func() {
						It("removes the layer, and its parent", func() {
							Expect(gc.GC(logger, fakeCake)).To(Succeed())

							Expect(fakeCake.RemoveCallCount()).To(Equal(2))
							Expect(fakeCake.RemoveArgsForCall(0)).To(Equal(layercake.DockerImageID("child")))
							Expect(fakeCake.RemoveArgsForCall(1)).To(Equal(layercake.DockerImageID("parent")))
						})
					})

					Context("when removing fails", func() {
						It("does not remove any more layers", func() {
							fakeCake.RemoveReturns(errors.New("cake failure"))
							gc.GC(logger, fakeCake)
							Expect(fakeCake.RemoveCallCount()).To(Equal(1))
						})
					})

					Context("but the layer has another child", func() {
						BeforeEach(func() {
							child2parent[layercake.DockerImageID("some-other-child")] = layercake.DockerImageID("parent")
						})

						It("removes only the initial layer", func() {
							child2parent[layercake.DockerImageID("child")] = layercake.DockerImageID("parent")
							Expect(gc.GC(logger, fakeCake)).To(Succeed())

							Expect(fakeCake.RemoveCallCount()).To(Equal(1))
							Expect(fakeCake.RemoveArgsForCall(0)).To(Equal(layercake.DockerImageID("child")))
						})
					})
				})

				Context("when the layer has grandparents", func() {
					It("removes all the grandparents", func() {
						child2parent[layercake.DockerImageID("child")] = layercake.DockerImageID("parent")
						child2parent[layercake.DockerImageID("parent")] = layercake.DockerImageID("granddaddy")

						Expect(gc.GC(logger, fakeCake)).To(Succeed())

						Expect(fakeCake.RemoveCallCount()).To(Equal(3))
						Expect(fakeCake.RemoveArgsForCall(0)).To(Equal(layercake.DockerImageID("child")))
						Expect(fakeCake.RemoveArgsForCall(1)).To(Equal(layercake.DockerImageID("parent")))
						Expect(fakeCake.RemoveArgsForCall(2)).To(Equal(layercake.DockerImageID("granddaddy")))
					})
				})
			})

			Context("when there are multiple leaves", func() {
				BeforeEach(func() {
					fakeCake.GetAllLeavesReturns([]layercake.ID{layercake.DockerImageID("child1"), layercake.DockerImageID("child2")}, nil)
				})

				It("removes all of the leaves", func() {
					Expect(gc.GC(logger, fakeCake)).To(Succeed())
					Expect(fakeCake.RemoveCallCount()).To(Equal(2))
					Expect(fakeCake.RemoveArgsForCall(0)).To(Equal(layercake.DockerImageID("child1")))
					Expect(fakeCake.RemoveArgsForCall(1)).To(Equal(layercake.DockerImageID("child2")))
				})

			})

			Context("when getting the list of leaves fails", func() {
				It("returns the error", func() {
					fakeCake.GetAllLeavesReturns(nil, errors.New("firey potato"))
					Expect(gc.GC(logger, fakeCake)).To(MatchError("firey potato"))
				})
			})
		})
	})

	Context("when the threshold is not exceeded", func() {
		BeforeEach(func() {
			fakeCake.GetAllLeavesReturns([]layercake.ID{layercake.DockerImageID("child1"), layercake.DockerImageID("child2")}, nil)
			graphCleanupThreshold = 1024
		})

		It("it does not clean up anything", func() {
			Expect(gc.GC(logger, fakeCake)).To(Succeed())
			Expect(fakeCake.RemoveCallCount()).To(Equal(0))
		})
	})

	Context("when image cleanup is disabled", func() {
		BeforeEach(func() {
			fakeCake.GetAllLeavesReturns([]layercake.ID{layercake.DockerImageID("child1"), layercake.DockerImageID("child2")}, nil)
			graphCleanupThreshold = -1
		})

		It("it does not clean up anything", func() {
			Expect(gc.GC(logger, fakeCake)).To(Succeed())
			Expect(fakeCake.RemoveCallCount()).To(Equal(0))
		})
	})

})
