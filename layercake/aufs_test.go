package layercake_test

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/cloudfoundry-incubator/garden-shed/layercake"
	"github.com/cloudfoundry-incubator/garden-shed/layercake/fake_cake"
	"github.com/cloudfoundry/gunk/command_runner/fake_command_runner"
	. "github.com/cloudfoundry/gunk/command_runner/fake_command_runner/matchers"
	"github.com/docker/docker/image"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Aufs", func() {
	var (
		runner   *fake_command_runner.FakeCommandRunner
		cake     *fake_cake.FakeCake
		aufsCake *layercake.AufsCake
	)

	BeforeEach(func() {
		runner = fake_command_runner.New()
		cake = new(fake_cake.FakeCake)

		aufsCake = &layercake.AufsCake{
			Cake:     cake,
			RootPath: "/path/to/banana",
			Runner:   runner,
			Logger:   lagertest.NewTestLogger("test"),
		}
	})

	Describe("MountNamespaced", func() {
		var id layercake.ID

		BeforeEach(func() {
			id = layercake.NamespacedLayerID{
				LayerID:  layercake.DockerImageID("blah"),
				CacheKey: "banana",
			}

			cake.GetReturns(&image.Image{
				ID:     id.GraphID(),
				Parent: "bananas-parent",
			}, nil)
		})

		It("should return the path to the layer", func() {
			path, err := aufsCake.MountNamespaced(id)
			Expect(err).NotTo(HaveOccurred())

			Expect(path).To(Equal(fmt.Sprintf("/path/to/banana/diff/%s", id.GraphID())))
		})

		Context("when getting image fails", func() {
			BeforeEach(func() {
				cake.GetReturns(nil, errors.New("oh no"))
			})

			It("returns the error", func() {
				_, err := aufsCake.MountNamespaced(id)
				Expect(err).To(MatchError("oh no"))
			})

			It("does not mount the parent layer", func() {
				aufsCake.MountNamespaced(id)

				Expect(cake.MountCallCount()).To(Equal(0))
			})
		})

		It("should mount the parent layer", func() {
			_, err := aufsCake.MountNamespaced(id)
			Expect(err).NotTo(HaveOccurred())

			Expect(cake.MountCallCount()).To(Equal(1))
			Expect(cake.MountArgsForCall(0)).To(Equal(layercake.DockerImageID("bananas-parent")))
		})

		Context("when the mount fails", func() {
			BeforeEach(func() {
				cake.MountReturns("", errors.New("mount did not work"))
			})

			It("should return an error", func() {
				_, err := aufsCake.MountNamespaced(id)
				Expect(err).To(MatchError("mount did not work"))
			})

			It("it should not copy the contents of the parent to the layer", func() {
				aufsCake.MountNamespaced(id)

				Expect(runner).NotTo(HaveExecutedSerially(fake_command_runner.CommandSpec{}))
			})
		})

		It("should copy the contents of the parent to the layer", func() {
			cake.MountReturns("/path/to/my/banana/layer", nil)

			_, err := aufsCake.MountNamespaced(id)
			Expect(err).NotTo(HaveOccurred())

			Expect(runner).To(HaveExecutedSerially(fake_command_runner.CommandSpec{
				Path: "sh",
				Args: []string{
					"-c",
					fmt.Sprintf("cp -a /path/to/my/banana/layer/* /path/to/banana/diff/%s", id.GraphID()),
				},
			}))
		})

		Context("when copying the contents fails", func() {
			It("should return the error", func() {
				runner.WhenRunning(
					fake_command_runner.CommandSpec{Path: "sh"},
					func(_ *exec.Cmd) error {
						return errors.New("I lost my banana!")
					},
				)

				_, err := aufsCake.MountNamespaced(id)
				Expect(err).To(MatchError("I lost my banana!"))
			})
		})
	})
})
