package rootfs_provider

import (
	"errors"

	"github.com/cloudfoundry-incubator/garden"
	"github.com/pivotal-golang/lager"
)

type BindMounter struct{}

func (b *BindMounter) Bind(log lager.Logger, rootFsPath string, bindSpec garden.BindMount) error {
	return errors.New("not implemented")
}
