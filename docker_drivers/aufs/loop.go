package aufs

import (
	"fmt"
	"io/ioutil"
	"os/exec"

	"code.cloudfoundry.org/lager"
)

type Loop struct {
	Retrier Retrier
	Logger  lager.Logger
}

func (lm *Loop) MountFile(filePath, destPath string) error {
	log := lm.Logger.Session("mount-file", lager.Data{"filePath": filePath, "destPath": destPath})
	log.Info(fmt.Sprintf("Start mounting %s on %s", filePath, destPath))
	defer log.Info(fmt.Sprintf("End mounting %s on %s", filePath, destPath))

	output, err := exec.Command("mount", "-n", "-t", "ext4", "-o", "loop,noatime",
		filePath, destPath).CombinedOutput()
	if err != nil {
		log.Error("mounting", err, lager.Data{"output": string(output)})
		return fmt.Errorf("mounting file: %s", err)
	}

	return nil
}

func (lm *Loop) Unmount(path string) error {
	contents, _ := ioutil.ReadFile("/proc/mounts")
	log := lm.Logger.Session("unmount", lager.Data{"path": path})
	log.Info(fmt.Sprintf("Start unmounting %s, mounts are: %s", path, contents))
	defer log.Info(fmt.Sprintf("End unmounting %s", path))

	var output []byte
	err := lm.Retrier.Run(func() error {
		var err error
		output, err = exec.Command("umount", "-d", path).CombinedOutput()
		if err != nil {
			if err2 := exec.Command("mountpoint", path).Run(); err2 != nil {
				// if it's not a mountpoint then this is fine
				return nil
			}
			log.Error(fmt.Sprintf("Error unmounting %s", path), err)
			return err
		}
		return nil
	})

	if err != nil {
		log.Error("unmounting", err, lager.Data{"output": string(output)})
		return fmt.Errorf("unmounting file (%s): %s", path, err)
	}

	return nil
}
