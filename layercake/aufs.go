package layercake

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry/gunk/command_runner"
	"github.com/pivotal-golang/lager"
)

type AufsCake struct {
	Cake

	RootPath string
	Runner   command_runner.CommandRunner

	Logger lager.Logger
}

// get the path of the directory
// copy files over from parent
// FUTURE: create backing store, attach to loop device and mount loop device in the directory before copying
func (aufs *AufsCake) MountNamespaced(id ID) (string, error) {
	img, err := aufs.Cake.Get(id)
	if err != nil {
		return "", err
	}

	parentPath, err := aufs.Cake.Mount(DockerImageID(img.Parent))
	if err != nil {
		return "", err
	}

	cmd := exec.Command("sh", "-c", fmt.Sprintf(
		"cp -a %s/* %s", parentPath, aufs.diffPath(id),
	))
	err = aufs.Runner.Run(cmd)
	if err != nil {
		return "", err
	}

	return aufs.diffPath(id), nil
}

func (aufs *AufsCake) diffPath(id ID) string {
	return filepath.Join(aufs.RootPath, "diff", id.GraphID())
}

func (aufs *AufsCake) mntPath(id ID) string {
	return filepath.Join(aufs.RootPath, "mnt", id.GraphID())
}
