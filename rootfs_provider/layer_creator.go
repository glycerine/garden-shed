package rootfs_provider

import (
	"fmt"
	"os/exec"
	"sync"

	"github.com/cloudfoundry-incubator/garden-shed/layercake"
	"github.com/cloudfoundry-incubator/garden-shed/repository_fetcher"
)

type ContainerLayerCreator struct {
	graph         Graph
	volumeCreator VolumeCreator
	namespacer    Namespacer
	mutex         *sync.Mutex

	fallback RootFSProvider
}

func NewLayerCreator(
	graph Graph,
	volumeCreator VolumeCreator,
	namespacer Namespacer,
) *ContainerLayerCreator {
	return &ContainerLayerCreator{
		graph:         graph,
		volumeCreator: volumeCreator,
		namespacer:    namespacer,
		mutex:         &sync.Mutex{},
	}
}

func (provider *ContainerLayerCreator) Create(id string, parentImage *repository_fetcher.Image, shouldNamespace bool, quota int64) (string, []string, error) {
	var err error
	var imageID layercake.ID = layercake.DockerImageID(parentImage.ImageID)
	if shouldNamespace {
		provider.mutex.Lock()
		imageID, err = provider.namespace(imageID)
		provider.mutex.Unlock()
		if err != nil {
			return "", nil, err
		}
	}

	containerID := layercake.ContainerID(id)
	err = provider.graph.Create(containerID, imageID)
	if err != nil {
		return "", nil, err
	}

	rootPath, err := provider.graph.QuotaedPath(containerID)
	if err != nil {
		return "", nil, err
	}

	for _, v := range parentImage.Volumes {
		if err = provider.volumeCreator.Create(rootPath, v); err != nil {
			return "", nil, err
		}
	}

	return rootPath, parentImage.Env, nil
}

func (provider *ContainerLayerCreator) namespace(imageID layercake.ID) (layercake.ID, error) {
	namespacedImageID := layercake.NamespacedID(imageID, provider.namespacer.CacheKey())

	if _, err := provider.graph.Get(namespacedImageID); err != nil {
		if err := provider.createNamespacedLayer(namespacedImageID, imageID); err != nil {
			return nil, err
		}
	}

	return namespacedImageID, nil
}

func (provider *ContainerLayerCreator) createNamespacedLayer(id, parentId layercake.ID) error {
	// create empty base layer to copy image into
	// this works around aufs not layering permissions nicely
	if err := provider.graph.Create(id, layercake.DockerImageID("")); err != nil {
		return err
	}

	parent, err := provider.graph.Path(parentId)
	if err != nil {
		return err
	}

	squashed, err := provider.graph.QuotaedPath(id)
	if err != nil {
		return err
	}

	// copy the whole thing down, we have to do this to get everything new inodes because aufs
	output, err := exec.Command("sh", "-c", fmt.Sprintf("cp -a %s/* %s", parent, squashed)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to copy the rootfs for aufs (%s): %s", err, output)
	}

	return provider.namespacer.Namespace(squashed)
}

func (provider *ContainerLayerCreator) createLayer(id, parentId layercake.ID) (string, error) {
	errs := func(err error) (string, error) {
		return "", err
	}

	if err := provider.graph.Create(id, parentId); err != nil {
		return errs(err)
	}

	namespacedRootfs, err := provider.graph.QuotaedPath(id)
	if err != nil {
		return errs(err)
	}

	return namespacedRootfs, nil
}
