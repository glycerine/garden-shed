package layercake

import "github.com/cloudfoundry/gunk/command_runner"

type AufsCake struct {
	Cake
	Runner command_runner.CommandRunner
}

func (a *AufsCake) Create(childID, parentID ID) error {
	if _, ok := childID.(NamespacedLayerID); !ok {
		return a.Cake.Create(childID, parentID)
	}

	if err := a.Cake.Create(childID, DockerImageID("")); err != nil {
		return err
	}
	// something more a.Cake.Get(childID)
	//	sourcePath, err := a.Cake.Path(parentID)
	//	destinationPath := a.Cake.Path(childID)
	// cp -a source destination
	// the metadata
	return nil
}
