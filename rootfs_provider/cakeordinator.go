package rootfs_provider

import (
	"net/url"
	"sync"

	"github.com/cloudfoundry-incubator/garden-shed/layercake"
	"github.com/cloudfoundry-incubator/garden-shed/repository_fetcher"
	"github.com/hashicorp/go-multierror"
	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter . LayerCreator
type LayerCreator interface {
	Create(id string, parentImage *repository_fetcher.Image, spec Spec) (string, []string, error)
}

//go:generate counterfeiter . RepositoryFetcher
type RepositoryFetcher interface {
	Fetch(*url.URL, int64) (*repository_fetcher.Image, error)
}

// CakeOrdinator manages a cake, fetching layers as neccesary
type CakeOrdinator struct {
	mu sync.RWMutex

	cake         layercake.Cake
	fetcher      RepositoryFetcher
	layerCreator LayerCreator
	retainer     layercake.Retainer
}

// New creates a new cake-ordinator, there should only be one CakeOrdinator
// for a particular cake.
func NewCakeOrdinator(cake layercake.Cake, fetcher RepositoryFetcher, layerCreator LayerCreator, retainer layercake.Retainer) *CakeOrdinator {
	return &CakeOrdinator{
		cake:         cake,
		fetcher:      fetcher,
		layerCreator: layerCreator,
		retainer:     retainer,
	}
}

func (c *CakeOrdinator) Create(logger lager.Logger, id string, spec Spec) (string, []string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fetcherDiskQuota := spec.QuotaSize
	if spec.QuotaScope == QuotaScopeExclusive {
		fetcherDiskQuota = 0
	}
	image, err := c.fetcher.Fetch(spec.RootFS, fetcherDiskQuota)
	if err != nil {
		return "", nil, err
	}

	return c.layerCreator.Create(id, image, spec)
}

func (c *CakeOrdinator) Retain(logger lager.Logger, id layercake.ID) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.retainer.Retain(logger, id)
}

func (c *CakeOrdinator) Destroy(logger lager.Logger, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	logger = logger.Session("ordinator-destroy", lager.Data{"id": id})

	return c.cake.Remove(layercake.ContainerID(id))
}

func (c *CakeOrdinator) GC(logger lager.Logger) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	logger = logger.Session("ordinator-gc")
	logger.Info("start")

	leaves, err := c.cake.GetAllLeaves()
	if err != nil {
		return err
	}

	var multiErr error
	for _, leafID := range leaves {
		logger.Info("removing leaf:", lager.Data{
			"leafID": leafID,
		})

		err := c.cake.Remove(layercake.DockerImageID(leafID))
		if err != nil {
			logger.Error("failed removing", err)
			multiErr = multierror.Append(multiErr, err)
		}
	}

	return multiErr
}
