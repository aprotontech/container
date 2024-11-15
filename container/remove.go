package container

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/shirou/gopsutil/disk"
	"github.com/spf13/cobra"

	"aproton.tech/container/utils"
)

func ContainerRemoveCommand(cmd *cobra.Command, args []string) {
	cmap, err := getContainerMetasMap()
	utils.Assert(err)

	for _, c := range args {
		if cnt, ok := cmap[c]; ok {
			if utils.IsProcessExists(cnt.ProcessID, cnt.Command) {
				utils.PrintToConsole("Container %s is running, please stop it first\n", c)
				break
			}

			removeContainerMeta(cnt)
			if cnt.Overlay != nil {
				unmountOverlayFileSystem(cnt.Overlay)
			}
			if cnt.Sandbox != "" {
				os.RemoveAll(cnt.Sandbox)
			}
		} else {
			utils.PrintToConsole("No such container: %s\n", c)
			break
		}
	}
}

func unmountOverlayFileSystem(overlay *Overlay) {
	mountPoint := overlay.MountPoint

	if !filepath.IsAbs(overlay.MountPoint) {
		if mp, err := filepath.Abs(overlay.MountPoint); err == nil {
			mountPoint = mp
		}
	}

	ps, err := disk.Partitions(true)
	utils.Assert(err)

	for _, p := range ps {
		if p.Mountpoint == mountPoint {
			utils.Assert(syscall.Unmount(overlay.MountPoint, 0))
			break
		}
	}

	if overlay.Upper != "" {
		os.RemoveAll(overlay.Upper)
	}
	if overlay.Upper != "" {
		os.RemoveAll(overlay.Working)
	}
}
