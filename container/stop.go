package container

import (
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"aproton.tech/container/utils"
)

func ContainerStopCommand(cmd *cobra.Command, args []string) {
	cmap, err := getContainerMetasMap()
	utils.Assert(err)

	wg := &sync.WaitGroup{}
	for _, c := range args {
		if cnt, ok := cmap[c]; ok {
			if utils.IsProcessExists(cnt.ProcessID, cnt.Command) {
				wg.Add(1)
				go func(cnt *ContainerMeta) {
					defer wg.Done()
					stopContainer(cnt)
				}(cnt)
			}
		}
	}
	wg.Wait()
}

func stopContainer(cnt *ContainerMeta) error {
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		for utils.IsProcessExists(cnt.ProcessID, cnt.Command) {
			time.Sleep(500 * time.Millisecond)
		}
	}()
	syscall.Kill(cnt.ProcessID, syscall.SIGTERM)
	select {
	case <-time.After(5 * time.Second):
		syscall.Kill(-cnt.ProcessID, syscall.SIGKILL)
	case <-ch:
	}

	return nil
}
