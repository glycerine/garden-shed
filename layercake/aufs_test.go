package layercake_test

import (
	"errors"
	"path/filepath"

	"github.com/cloudfoundry-incubator/garden-shed/layercake"

	"io/ioutil"

	"github.com/cloudfoundry-incubator/garden-shed/layercake/fake_cake"
	"github.com/cloudfoundry-incubator/garden-shed/layercake/fake_id"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Aufs", func() {

	var (
		aufsCake          *layercake.AufsCake
		cake              *fake_cake.FakeCake
		parentID          *fake_id.FakeID
		childID           *fake_id.FakeID
		testError         error
		namespacedChildID layercake.ID
	)

	BeforeEach(func() {
		cake = new(fake_cake.FakeCake)
		aufsCake = &layercake.AufsCake{
			Cake: cake,
		}
		parentID = new(fake_id.FakeID)
		parentID.GraphIDReturns("graph-id")

		childID = new(fake_id.FakeID)
		testError = errors.New("bad")
		namespacedChildID = layercake.NamespacedID(parentID, "test")
	})

	Describe("DriverName", func() {
		BeforeEach(func() {
			cake.DriverNameReturns("driver-name")
		})
		It("should delegate to the cake", func() {
			dn := aufsCake.DriverName()
			Expect(cake.DriverNameCallCount()).To(Equal(1))
			Expect(dn).To(Equal("driver-name"))
		})
	})

	Describe("Create", func() {
		Context("when the child ID is namespaced", func() {
			It("should delegate to the cake but with an empty parent", func() {
				cake.CreateReturns(testError)
				Expect(aufsCake.Create(namespacedChildID, parentID)).To(Equal(testError))
				Expect(cake.CreateCallCount()).To(Equal(1))
				cid, iid := cake.CreateArgsForCall(0)
				Expect(cid).To(Equal(namespacedChildID))
				Expect(iid.GraphID()).To(BeEmpty())
			})

			It("should copy the parent layer to the child layer", func() {
				parentDir, err := ioutil.TempDir("", "parent-layer")
				Expect(err).NotTo(HaveOccurred())
				Expect(ioutil.WriteFile(filepath.Join(parentDir, "somefile"), []byte("somecontents"), 0755)).To(Succeed())

				cake.PathStub = func(id layercake.ID) (string, error) {
					if id == parentID {
						return parentDir, nil
					}
					return "", nil
				}
			})
		})

		Context("when the image ID is not namespaced", func() {
			It("should delegate to the cake", func() {
				cake.CreateReturns(testError)
				Expect(aufsCake.Create(childID, parentID)).To(Equal(testError))
				Expect(cake.CreateCallCount()).To(Equal(1))
				cid, iid := cake.CreateArgsForCall(0)
				Expect(cid).To(Equal(childID))
				Expect(iid).To(Equal(parentID))
			})
		})

	})

	Describe("Get", func() {

	})

	Describe("Remove", func() {

	})

	Describe("IsLeaf", func() {

	})
})
