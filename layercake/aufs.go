package layercake

import (
	"os"
	"os/exec"
	"path/filepath"

	"fmt"

	"github.com/cloudfoundry/gunk/command_runner"
)

const (
	metadataDirName    string = "garden-info"
	parentChildDirName string = "parent-child"
	childParentDirName string = "child-parent"
)

type AufsCake struct {
	Cake
	Runner    command_runner.CommandRunner
	GraphRoot string
}

func (a *AufsCake) Create(childID, parentID ID) error {
	if _, ok := childID.(NamespacedLayerID); !ok {
		return a.Cake.Create(childID, parentID)
	}

	if err := a.Cake.Create(childID, DockerImageID("")); err != nil {
		return err
	}

	_, err := a.Cake.Get(childID)
	if err != nil {
		return err
	}

	sourcePath, err := a.Cake.Path(parentID)
	if err != nil {
		return err
	}

	destinationPath, err := a.Cake.Path(childID)
	if err != nil {
		return err
	}

	copyCmd := fmt.Sprintf("cp -a %s/. %s", sourcePath, destinationPath)
	if err := a.Runner.Run(exec.Command("sh", "-c", copyCmd)); err != nil {
		return err
	}

	parentChildDir := filepath.Join(a.GraphRoot, metadataDirName, parentChildDirName)
	childParentDir := filepath.Join(a.GraphRoot, metadataDirName, childParentDirName)

	if err = a.writeInfo(parentChildDir, parentID, childID); err != nil {
		return err
	}

	if err = a.writeInfo(childParentDir, childID, parentID); err != nil {
		return err
	}

	return nil
}

func (a *AufsCake) IsLeaf(id ID) (bool, error) {
	if isDockerLeaf, err := a.Cake.IsLeaf(id); err != nil {
		return false, err
	} else if !isDockerLeaf {
		return false, nil
	}

	isParent, err := a.isParentOfNamespacedChild(
		filepath.Join(a.GraphRoot, metadataDirName, parentChildDirName),
		id,
	)

	if err != nil {
		return false, err
	}

	return !isParent, nil
}

func (a *AufsCake) isParentOfNamespacedChild(path string, id ID) (bool, error) {
	if _, err := os.Stat(filepath.Join(path, id.GraphID())); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (a *AufsCake) writeInfo(path string, file ID, content ID) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	handle, err := os.OpenFile(
		filepath.Join(path, file.GraphID()),
		os.O_CREATE|os.O_RDWR|os.O_APPEND,
		0755)
	if err != nil {
		return err
	}
	defer handle.Close()

	fmt.Fprintln(handle, content.GraphID())

	return nil
}
