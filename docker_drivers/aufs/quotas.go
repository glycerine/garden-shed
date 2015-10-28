package aufs

import (
	"crypto/sha256"
	"fmt"
)

//go:generate counterfeiter . LoopMounter
type LoopMounter interface {
	MountFile(filePath, destPath string) error
	Unmount(path string) error
}

//go:generate counterfeiter . BackingStoreMgr
type BackingStoreMgr interface {
	Create(id string, quota int64) (string, error)
	Delete(id string) error
}

type QuotaLayer struct {
	BackingStoreMgr BackingStoreMgr
	LoopMounter     LoopMounter
}

func (q *QuotaLayer) Provide(dest string, quota int64) error {
	bsPath, err := q.BackingStoreMgr.Create(q.hash(dest), quota)
	if err != nil {
		return err
	}

	return q.LoopMounter.MountFile(bsPath, dest)
}

func (q *QuotaLayer) Destroy(dest string) error {
	if err := q.LoopMounter.Unmount(dest); err != nil {
		return err
	}

	return q.BackingStoreMgr.Delete(q.hash(dest))
}

func (q *QuotaLayer) hash(id string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(id)))
}
