/*

aufs driver directory structure

.
├── layers // Metadata of layers
│   ├── 1
│   ├── 2
│   └── 3
├── diff  // Content of the layer
│   ├── 1  // Contains layers that need to be mounted for the id
│   ├── 2
│   └── 3
└── mnt    // Mount points for the rw layers to be mounted
    ├── 1
    ├── 2
    └── 3

*/

package aufs

import (
	"path/filepath"

	"github.com/docker/docker/daemon/graphdriver"
)

//go:generate counterfeiter . GraphDriver
type GraphDriver interface {
	graphdriver.Driver
}

//go:generate counterfeiter . QuotaLayerProvider
type QuotaLayerProvider interface {
	Provide(dest string, quota int64) error
	Destroy(dest string) error
}

type Driver struct {
	GraphDriver
	QuotaLayerProvider QuotaLayerProvider
	RootPath           string
}

func (a *Driver) GetQuotaed(id, mountlabel string, quota int64) (string, error) {
	path := filepath.Join(a.RootPath, "aufs", "diff", id)
	if err := a.QuotaLayerProvider.Provide(path, quota); err != nil {
		return "", err
	}

	return a.GraphDriver.Get(id, mountlabel)
}

func (a *Driver) RemoveQuotaed(id string) error {
	path := filepath.Join(a.RootPath, "aufs", "diff", id)
	if err := a.QuotaLayerProvider.Destroy(path); err != nil {
		return err
	}

	return a.GraphDriver.Remove(id)
}
