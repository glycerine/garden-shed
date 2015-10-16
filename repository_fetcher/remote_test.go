package repository_fetcher_test

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/url"

	"github.com/docker/docker/image"

	"github.com/cloudfoundry-incubator/garden-shed/distclient"
	"github.com/cloudfoundry-incubator/garden-shed/distclient/fake_distclient"
	"github.com/cloudfoundry-incubator/garden-shed/layercake"
	"github.com/cloudfoundry-incubator/garden-shed/layercake/fake_cake"
	"github.com/cloudfoundry-incubator/garden-shed/repository_fetcher"
	"github.com/cloudfoundry-incubator/garden-shed/repository_fetcher/fakes"
	"github.com/docker/distribution/digest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Fetching from a Remote repo", func() {
	var (
		fakeDialer *fakes.FakeDialer
		fakeConn   *fake_distclient.FakeConn
		fakeCake   *fake_cake.FakeCake
		remote     *repository_fetcher.Remote

		manifests      map[string]*distclient.Manifest
		blobs          map[digest.Digest]string
		existingLayers map[string]bool
	)

	BeforeEach(func() {
		existingLayers = map[string]bool{}

		manifests = map[string]*distclient.Manifest{
			"latest": &distclient.Manifest{
				Layers: []distclient.Layer{
					{},
				}},
			"some-tag": &distclient.Manifest{
				Layers: []distclient.Layer{
					{Digest: "abc-def", ID: "abc-id", Parent: "abc-parent-id"},
					{"ghj-klm", "ghj-id", ""},
				},
			},
		}

		blobs = map[digest.Digest]string{
			"abc-def": "abc-def-contents",
			"ghj-klm": "ghj-klm-contents",
		}

		fakeConn = new(fake_distclient.FakeConn)
		fakeConn.GetManifestStub = func(_ lager.Logger, tag string) (*distclient.Manifest, error) {
			return manifests[tag], nil
		}

		fakeConn.GetBlobReaderStub = func(_ lager.Logger, digest digest.Digest) (io.Reader, error) {
			return bytes.NewReader([]byte(blobs[digest])), nil
		}

		fakeDialer = new(fakes.FakeDialer)
		fakeDialer.DialStub = func(_ lager.Logger, host, repo string) (distclient.Conn, error) {
			return fakeConn, nil
		}

		fakeCake = new(fake_cake.FakeCake)
		fakeCake.GetStub = func(id layercake.ID) (*image.Image, error) {
			if _, ok := existingLayers[id.GraphID()]; ok {
				return &image.Image{}, nil
			}

			return nil, errors.New("doesnt exist")
		}

		remote = repository_fetcher.NewRemote(lagertest.NewTestLogger("test"), "the-default-host", fakeCake, fakeDialer)
	})

	Context("when the URL has a host", func() {
		It("dials that host over https", func() {
			_, err := remote.Fetch(parseURL("docker://some-host/some/repo#some-tag"), 1234)
			Expect(err).NotTo(HaveOccurred())

			_, host, _ := fakeDialer.DialArgsForCall(0)
			Expect(host).To(Equal("https://some-host"))
		})
	})

	Context("when the host is empty", func() {
		It("uses the default host", func() {
			_, err := remote.Fetch(parseURL("docker:///some/repo#some-tag"), 1234)
			Expect(err).NotTo(HaveOccurred())

			_, host, _ := fakeDialer.DialArgsForCall(0)
			Expect(host).To(Equal("https://the-default-host"))
		})
	})

	Context("when the path contains a slash", func() {
		It("uses the path explicitly", func() {
			_, err := remote.Fetch(parseURL("docker://some-host/some/repo#some-tag"), 1234)
			Expect(err).NotTo(HaveOccurred())

			_, _, repo := fakeDialer.DialArgsForCall(0)
			Expect(repo).To(Equal("some/repo"))
		})
	})

	Context("when the path does not contain a slash", func() {
		It("preprends the implied 'library/' to the path", func() {
			_, err := remote.Fetch(parseURL("docker://some-host/somerepo#some-tag"), 1234)
			Expect(err).NotTo(HaveOccurred())

			_, _, repo := fakeDialer.DialArgsForCall(0)
			Expect(repo).To(Equal("library/somerepo"))
		})
	})

	Context("when the cake does not contain any of the layers", func() {
		It("registers each of the layers in the graph", func() {
			_, err := remote.Fetch(parseURL("docker:///foo#some-tag"), 64)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeCake.RegisterCallCount()).To(Equal(2))
		})

		It("registers the right layer contents", func() {
			_, err := remote.Fetch(parseURL("docker:///foo#some-tag"), 64)
			Expect(err).NotTo(HaveOccurred())

			image, reader := fakeCake.RegisterArgsForCall(0)
			Expect(image.ID).To(Equal("abc-id"))
			Expect(image.Parent).To(Equal("abc-parent-id"))

			b, err := ioutil.ReadAll(reader)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(b)).To(Equal("abc-def-contents"))
		})
	})

	Context("when the graph already contains a layer", func() {
		BeforeEach(func() {
			existingLayers["ghj-id"] = true
		})

		It("avoids registering it again", func() {
			_, err := remote.Fetch(parseURL("docker:///foo#some-tag"), 64)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeCake.RegisterCallCount()).To(Equal(1))
		})
	})

	Context("when the url doesnot contain a fragment", func() {
		It("uses 'latest' as the tag", func() {
			_, err := remote.Fetch(parseURL("docker:///foo"), 64)
			Expect(err).NotTo(HaveOccurred())

			_, tag := fakeConn.GetManifestArgsForCall(0)
			Expect(tag).To(Equal("latest"))
		})
	})

	It("returns an image with the ID of the top layer", func() {
		img, _ := remote.Fetch(parseURL("docker:///foo#some-tag"), 64)
		Expect(img.ImageID).To(Equal("ghj-id"))
	})

	It("can fetch just the ID", func() {
		id, _ := remote.FetchID(parseURL("docker:///foo#some-tag"))
		Expect(id).To(Equal(layercake.DockerImageID("ghj-id")))
	})
})

func parseURL(u string) *url.URL {
	r, err := url.Parse(u)
	Expect(err).NotTo(HaveOccurred())

	return r
}
