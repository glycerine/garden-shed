package quota

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/pivotal-golang/lager"
)

type Enforcer interface {
	Enforce(logger lager.Logger, directory string, limit uint64) error
	Unenforce(logger lager.Logger, directory string) error
}

type xfs struct {
	root string
}

func NewXFSEnforcer(root string) Enforcer {
	return &xfs{root: root}
}

func (x *xfs) Enforce(logger lager.Logger, dir string, limit uint64) error {
	projects, err := os.OpenFile("/etc/projects", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	projids, err := os.OpenFile("/etc/projid", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	_, err = projects.WriteString("1:" + dir)
	if err != nil {
		return err
	}

	_, err = projids.WriteString("someprojid:1")
	if err != nil {
		return err
	}

	projects.Close()
	projids.Close()

	out, err := exec.Command("xfs_quota", "-x", "-c", "project -s someprojid").CombinedOutput()
	if err != nil {
		return fmt.Errorf("xfs quota setup: %s", out)
	}

	out, err = exec.Command("xfs_quota", "-x", "-c", "limit -p bhard=1000 someprojid").CombinedOutput()
	if err != nil {
		return fmt.Errorf("xfs quota setup: %s", out)
	}

	return nil
}

func (x *xfs) Unenforce(logger lager.Logger, dir string) error {
	return nil
}
