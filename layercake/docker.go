// Abstracts a layered filesystem provider, such as docker's ImageStore
package layercake

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/docker/distribution/digest"
	"github.com/docker/docker/daemon/graphdriver"
	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/archive"
)

type QuotaedDriver interface {
	graphdriver.Driver
	GetQuotaed(id, mountlabel string, quota int64) (string, error)
}

type Docker struct {
	ImageStore image.Store
	Driver     graphdriver.Driver
}

func (d *Docker) DriverName() string {
	return d.Driver.String()
}

func (d *Docker) Create(layerID, parentID ID, containerID string) error {
	return d.Register(
		&image.Image{
			V1Image: image.V1Image{ID: layerID.GraphID(), Container: containerID, Parent: parentID.GraphID()},
			Parent:  image.ID(digest.Digest(parentID.GraphID())),
		}, nil)
}

func (d *Docker) Register(image *image.Image, layer archive.Reader) error {
	return d.ImageStore.Register(&descriptor{image}, layer)
}

func (d *Docker) Get(id ID) (*image.Image, error) {
	return d.ImageStore.Get(image.ID(digest.Digest(id.GraphID())))
}

func (d *Docker) Unmount(id ID) error {
	return d.Driver.Put(id.GraphID())
}

func (d *Docker) Remove(id ID) error {
	if err := d.Driver.Put(id.GraphID()); err != nil {
		return err
	}

	_, err := d.ImageStore.Delete(image.ID(digest.Digest(id.GraphID())))
	return err
}

func (d *Docker) Path(id ID) (string, error) {
	return d.Driver.Get(id.GraphID(), "")
}

func (d *Docker) QuotaedPath(id ID, quota int64) (string, error) {
	if d.DriverName() == "aufs" {
		return d.Driver.(QuotaedDriver).GetQuotaed(id.GraphID(), "", quota)
	} else {
		return "", errors.New("quotas are not supported for this driver")
	}
}

func (d *Docker) All() (layers []*image.Image) {
	for _, layer := range d.ImageStore.Map() {
		layers = append(layers, layer)
	}
	return layers
}

func (d *Docker) IsLeaf(id ID) (bool, error) {
	heads := d.ImageStore.Heads()
	_, ok := heads[image.ID(digest.Digest(id.GraphID()))]
	return ok, nil
}

func (d *Docker) GetAllLeaves() ([]ID, error) {
	heads := d.ImageStore.Heads()
	var result []ID

	for head := range heads {
		result = append(result, DockerImageID(head))
	}

	return result, nil
}

type ContainerID string
type DockerImageID string

type LocalImageID struct {
	Path         string
	ModifiedTime time.Time
}

type NamespacedLayerID struct {
	LayerID  ID
	CacheKey string
}

func NamespacedID(id ID, cacheKey string) NamespacedLayerID {
	return NamespacedLayerID{id, cacheKey}
}

func (c ContainerID) GraphID() string {
	return shaID(string(c))
}

func (d DockerImageID) GraphID() string {
	return string(d)
}

func (c LocalImageID) GraphID() string {
	return shaID(fmt.Sprintf("%s-%d", c.Path, c.ModifiedTime))
}

func (n NamespacedLayerID) GraphID() string {
	return shaID(n.LayerID.GraphID() + "@" + n.CacheKey)
}

func shaID(id string) string {
	if id == "" {
		return id
	}

	return fmt.Sprintf("%x", sha256.Sum256([]byte(id)))
}

type descriptor struct {
	image *image.Image
}

func (d descriptor) ID() string {
	return d.image.V1Image.ID
}

func (d descriptor) Parent() string {
	return d.image.V1Image.Parent
}

func (d descriptor) MarshalConfig() ([]byte, error) {
	return json.Marshal(d.image)
}
