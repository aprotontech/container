package container

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"aproton.tech/container/utils"
)

const CgroupPathPrefix = "/sys/fs/cgroup/container.slice/"

type SetLimit func(containerId string) error

func SetContainerCgroup(containerId string, setter ...SetLimit) {
	initCgroup(containerId)
	for _, s := range setter {
		utils.Assert(s(containerId))
	}
}

func RemoveContainerCgroup(containerId string) {
	utils.Assert(os.RemoveAll(getContainerCGroupPath(containerId)))
}

func SetMaxMemory(maxMemory uint64) SetLimit {
	return func(containerId string) error {
		return os.WriteFile(getContainerCGroupPath(containerId, "memory.max"), []byte(fmt.Sprintf("%d", maxMemory)), 0644)
	}
}

func SetProcessId(pid int) SetLimit {
	return func(containerId string) error {
		return os.WriteFile(getContainerCGroupPath(containerId, "cgroup.procs"), []byte(fmt.Sprintf("%d", pid)), 0644)
	}
}

func getContainerCGroupPath(containerId string, subfile ...string) string {
	s := filepath.Join(CgroupPathPrefix, containerId+".scope")
	for _, f := range subfile {
		s = filepath.Join(s, f)
	}
	return s
}

func initCgroup(containerId string) {
	ccpath := getContainerCGroupPath(containerId)
	_, err := os.Stat(ccpath)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		utils.Assert(err)
	}

	utils.Assert(os.MkdirAll(ccpath, 0755))

	isControllerOK := func(controllerspath string) bool {
		content, err := os.ReadFile(controllerspath)
		utils.Assert(err)
		return strings.Contains(string(content), "cpu") && strings.Contains(string(content), "memory")
	}

	if !isControllerOK(filepath.Join(ccpath, "cgroup.controllers")) {
		logrus.Infof("not found cpu/memory in controller")
		if !isControllerOK("/sys/fs/cgroup/cgroup.subtree_control") {
			content, err := os.ReadFile("/sys/fs/cgroup/cgroup.procs")
			utils.Assert(err)
			for _, pid := range strings.Split(string(content), "\n") {
				pid = strings.Trim(pid, " ")
				if pid != "" {
					os.WriteFile(filepath.Join(CgroupPathPrefix, "cgroup.procs"), []byte(pid), 0644)
				}
			}

			content, err = os.ReadFile("/sys/fs/cgroup/cgroup.controllers")
			utils.Assert(err)
			for _, ctrl := range strings.Split(string(content), " ") {
				if ctrl != "" {
					utils.Assert(os.WriteFile("/sys/fs/cgroup/cgroup.subtree_control", []byte("+"+ctrl), 0644))
				}
			}
		}

		if !isControllerOK(filepath.Join(CgroupPathPrefix, "cgroup.subtree_control")) {
			logrus.Infof("append subtree_control")
			content, err := os.ReadFile(filepath.Join(CgroupPathPrefix, "cgroup.controllers"))
			utils.Assert(err)
			for _, ctrl := range strings.Split(string(content), " ") {
				if ctrl != "" {
					utils.Assert(os.WriteFile(filepath.Join(CgroupPathPrefix, "cgroup.subtree_control"), []byte("+"+ctrl), 0644))
				}
			}
		}

	}
}
