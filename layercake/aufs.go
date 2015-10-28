package layercake

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"fmt"

	"github.com/cloudfoundry/gunk/command_runner"
	"github.com/docker/docker/image"
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

	if isAlreadyNamespaced, err := a.hasInfo(a.childParentDir(), childID); err != nil {
		return err
	} else if isAlreadyNamespaced {
		return fmt.Errorf("%s already exists", childID.GraphID())
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

	if err = a.writeInfo(a.parentChildDir(), parentID, childID); err != nil {
		return err
	}

	if err = a.writeInfo(a.childParentDir(), childID, parentID); err != nil {
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

	isParent, err := a.hasInfo(a.parentChildDir(), id)
	if err != nil {
		return false, err
	}

	return !isParent, nil
}

func (a *AufsCake) Get(id ID) (*image.Image, error) {
	img, err := a.Cake.Get(id)
	if err != nil {
		return nil, err
	}

	if img.Parent == "" {
		parentData, err := ioutil.ReadFile(filepath.Join(a.childParentDir(), id.GraphID()))
		if err != nil {
			if os.IsNotExist(err) {
				return img, nil
			}
			return nil, err
		}

		img.Parent = strings.TrimSpace(string(parentData))
	}
	return img, nil
}

func (a *AufsCake) Remove(id ID) error {
	return a.Cake.Remove(id)
}

func (a *AufsCake) hasInfo(path string, id ID) (bool, error) {
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

func (a *AufsCake) parentChildDir() string {
	return filepath.Join(a.GraphRoot, metadataDirName, parentChildDirName)
}

func (a *AufsCake) childParentDir() string {
	return filepath.Join(a.GraphRoot, metadataDirName, childParentDirName)
}
