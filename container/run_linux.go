//go:build linux

package container

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"aproton.tech/container/image"
	"aproton.tech/container/utils"
	"github.com/dustin/go-humanize"
	"github.com/go-faker/faker/v4"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/lithammer/shortuuid"
	"github.com/moby/moby/pkg/reexec"
	"github.com/shirou/gopsutil/disk"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const upperPath = "var/overlay/upper"
const WorkingPath = "var/overlay/working"
const CAP_SYS_ADMIN uint64 = 1 << 21

func ContainerRunCommand(cmd *cobra.Command, args []string) {
	imgname, err := name.ParseReference(args[0])
	utils.Assert(err)
	img, err := image.GetImage(imgname.Name(), false)
	utils.Assert(err)

	containerId := shortuuid.New()

	var sdx *Overlay
	var sandbox string
	if canUseOverlay() {
		sdx = buildOverlaySandbox(img, containerId)
		sandbox = sdx.MountPoint
	} else {
		sandbox = image.BuildSandbox(img, containerId)
	}

	config, _ := image.GetImageConfig(img)

	cnt := buildProcessCmd(config, args[1:])

	os.MkdirAll("var/runtime", 0755)
	tmpFile := fmt.Sprintf("var/runtime/%s.json", containerId)

	if err := os.WriteFile(tmpFile, cnt, 440); err != nil {
		utils.Assert(err)
	}

	defer os.Remove(tmpFile)

	childcmd := reexec.Command(ReExecRunCommand, sandbox, tmpFile)

	childcmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Cloneflags: syscall.CLONE_NEWNS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWPID,
	}

	if cmd.Flag("tty") != nil && cmd.Flag("tty").Value.String() == "true" {
		if cmd.Flag("interactive") != nil || cmd.Flag("interactive").Value.String() == "true" {
			childcmd.Stdin = os.Stdin
			childcmd.SysProcAttr.Foreground = true
		}

		childcmd.Stdout = os.Stdout
		childcmd.Stderr = os.Stderr
	}

	cntMeta := &ContainerMeta{
		Name:        faker.Username(),
		ProcessID:   0,
		Image:       imgname.Name(),
		ContainerID: containerId,
		Created:     time.Now(),
		Command:     config.Cmd[0],
		Status:      "Exit",
		Ports:       "",
		Sandbox:     sandbox,
		Overlay:     sdx,
	}

	if cmd.Flag("rm") != nil && cmd.Flag("rm").Value.String() == "true" {
		//defer
	}

	appendContainerMeta(cntMeta)

	defer RemoveContainerCgroup(containerId)

	if cmd.Flag("memory") != nil && cmd.Flag("memory").Value.String() != "" {
		size, err := humanize.ParseBytes(cmd.Flag("memory").Value.String())
		logrus.Infof("memory = %d", size)
		utils.Assert(err)
		SetContainerCgroup(containerId, SetMaxMemory(size))
	}

	utils.Assert(childcmd.Start(), "start failed with error ")

	SetContainerCgroup(containerId, SetProcessId(childcmd.Process.Pid))

	cntMeta.ProcessID = childcmd.Process.Pid
	cntMeta.Status = "RUNNING"
	updateContainerMeta(cntMeta)

	if err := childcmd.Wait(); err != nil {
		if !strings.Contains(err.Error(), "exit status") {
			utils.Assert(err)
		}
	}
}

func Run(sandbox, cmdpath string) error {
	cnt, err := os.ReadFile(cmdpath)
	utils.Assert(err)

	logrus.Infof("mypid=%d", syscall.Getpid())

	logrus.Infof("Config=%s", string(cnt))

	var config v1.Config
	json.Unmarshal(cnt, &config)
	utils.Assert(err)

	utils.Assert(buildNetworkEnv(sandbox))

	utils.Assert(buildFileSystem(sandbox))

	if config.User != "" {
		utils.Assert(buildUser(config.User))
	}

	if config.WorkingDir == "" {
		config.WorkingDir = "/"
	}

	utils.Assert(syscall.Chdir(config.WorkingDir))

	logrus.Infof("++++++++++++++++++++++++++++++++++++++++++++++")
	logrus.Infof("WorkingDir(%s),Command(%s)", config.WorkingDir, strings.Join(config.Cmd, " "))

	return syscall.Exec(config.Cmd[0], config.Cmd, config.Env)
}

func buildOverlaySandbox(img v1.Image, containerId string) *Overlay {
	layers := image.FlatImageFiles(img)
	upper := filepath.Join(upperPath, containerId)
	utils.Assert(os.MkdirAll(upper, 0755))
	workpath := filepath.Join(WorkingPath, containerId)
	utils.Assert(os.MkdirAll(workpath, 0755))
	sandbox := filepath.Join(image.SandboxPath, containerId)
	utils.Assert(os.MkdirAll(sandbox, 0755))
	utils.Assert(syscall.Mount("overlay", sandbox, "overlay", 0,
		fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", strings.Join(layers, ":"), upper, workpath)))
	return &Overlay{
		Working:    workpath,
		Upper:      upper,
		MountPoint: sandbox,
	}
}

func buildNetworkEnv(sandbox string) error {
	hostname := shortuuid.New()
	if err := syscall.Sethostname([]byte(hostname)); err != nil {
		return err
	}

	hosts := []string{
		"127.0.0.1       localhost",
		"::1     localhost ip6-localhost ip6-loopback",
		"fe00::0 ip6-localnet",
		"ff00::0 ip6-mcastprefix",
		"ff02::1 ip6-allnodes",
		"ff02::2 ip6-allrouters",
		"172.17.0.2 " + hostname,
	}
	if err := os.WriteFile(filepath.Join(sandbox, "/etc/hosts"), []byte(strings.Join(hosts, "\n")), 644); err != nil {
		return err
	}

	content, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(sandbox, "/etc/resolv.conf"), content, 644); err != nil {
		return err
	}

	logrus.Infof("Hostname=%s", hostname)

	return nil
}

func buildFileSystem(sandbox string) error {
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE, ""); err != nil {
		return err
	}

	if err := syscall.Mount("proc", filepath.Join(sandbox, "/proc"), "proc", 0, ""); err != nil {
		return err
	}

	if err := syscall.Mount("dev", filepath.Join(sandbox, "/dev"), "devtmpfs", 0, ""); err != nil {
		return err
	}

	if err := syscall.Mount("sys", filepath.Join(sandbox, "/sys"), "sysfs", 0, ""); err != nil {
		return err
	}

	if err := syscall.Chroot(sandbox); err != nil {
		return err
	}

	return nil
}

func buildUser(uname string) error {
	u, err := user.Lookup(uname)
	if err == nil {
		setId := func(setter func(int) error, sid string) error {
			uid, err := strconv.Atoi(sid)
			if err != nil {
				return err
			}
			return setter(uid)
		}
		if u.Uid != "" {
			setId(syscall.Setuid, u.Uid)
		}
		if u.Gid != "" {
			setId(syscall.Setgid, u.Gid)
		}
	}

	return nil
}

func buildProcessCmd(config *v1.Config, cmds []string) []byte {
	args := []string{}
	if len(cmds) != 0 {
		args = cmds
	} else {
		if len(config.Entrypoint) != 0 {
			args = append(args, config.Entrypoint...)
		}
	}

	config.Cmd = args

	r, _ := json.Marshal(config)
	return r
}

func canUseOverlay() bool {
	partitions, err := disk.Partitions(true)
	utils.Assert(err)
	var matched *disk.PartitionStat
	for _, partition := range partitions {
		if strings.HasPrefix(image.SandboxPath, partition.Mountpoint) {
			if matched == nil || (len(matched.Mountpoint) < len(partition.Mountpoint)) {
				matched = &partition
			}
		}
	}

	if matched != nil && matched.Fstype == "overlay" {
		return false
	}

	file, err := os.Open("/proc/self/status")
	utils.Assert(err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "CapEff:\t") {
			cap, err := strconv.ParseUint(line[len("CapEff:\t"):], 16, 64)
			utils.Assert(err)
			return (cap & CAP_SYS_ADMIN) == CAP_SYS_ADMIN
		}
	}

	return true
}
