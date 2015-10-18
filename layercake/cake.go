package layercake

import (
	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/archive"
)

type ID interface {
	GraphID() string
}

//go:generate counterfeiter -o fake_cake/fake_cake.go . Cake
type Cake interface {
	DriverName() string
	Create(containerID, imageID ID) error
	Register(img *image.Image, layer archive.ArchiveReader) error
	Get(id ID) (*image.Image, error)
	Remove(id ID) error
	Path(id ID) (string, error)
	QuotaedPath(id ID) (string, error)
	IsLeaf(id ID) (bool, error)
}
