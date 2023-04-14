package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type runningContainerInfo struct {
	containerId string
	image       string
	command     string
	pid         int
}

// This isn't a great implementation and cna possibly be simplified
// using regex. But for now, here we are. Thsi function gets the
// current mount points, figures out which image is mounted for a
// given container ID, looks it up in out images database which we
// maintain and returns teh image and tag information.

func getDistribution(containerID string) (string, error) {
	var lines []string
	file, err := os.Open("/proc/mounts")
	if err != nil {
		fmt.Println("Unable to read /proc/mounts")
		return "", err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	for _, line := range lines {
		if strings.Contains(line, containerID) {
			parts := strings.Split(line, " ")
			for _, part := range parts {
				if strings.Contains(part, "lowerdir=") {
					options := strings.Split(part, ",")
					for _, option := range options {
						if strings.Contains(option, "lowerdir=") {
							imagesPath := getGockerImagesPath()
							leaderString := "lowerdir=" + imagesPath + "/"
							trailerString := option[len(leaderString):]
							imageID := trailerString[:12]
							image, tag := getImageAndTagForHash(imageID)
							return fmt.Sprintf("%s:%s", image, tag), nil
						}
					}
				}
			}
		}
	}
	return "", nil
}

func getRunningContainerInfoForId(containerID string) (runningContainerInfo, error) {
	container := runningContainerInfo{}
	var procs []string
	basePath := "/sys/fs/cgroup/cpu/gocker"

	file, err := os.Open(basePath + "/" + containerID + "/cgroup.procs")
	if err != nil {
		fmt.Println("Unable to read cgroup.procs")
		return container, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		procs = append(procs, scanner.Text())
	}
	if len(procs) > 0 {
		pid, err := strconv.Atoi(procs[len(procs)-1])
		if err != nil {
			fmt.Println("Unable to read PID")
			return container, err
		}
		cmd, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
		containerMntPath := getGockerContainersPath() + "/" + containerID + "/fs/mnt"
		realContainerMntPath, err := filepath.EvalSymlinks(containerMntPath)
		if err != nil {
			fmt.Println("Unable to resolve path")
			return container, err
		}

		image, _ := getDistribution(containerID)
		container = runningContainerInfo{
			containerId: containerID,
			image:       image,
			command:     cmd[len(realContainerMntPath):],
			pid:         pid,
		}
	}
	return container, nil
}

func getRunningContainers() ([]runningContainerInfo, error) {
	var containers []runningContainerInfo
	basePath := "/sys/fs/cgroup/cpu/gocker"

	entries, err := os.ReadDir(basePath)
	if os.IsNotExist(err) {
		return containers, nil
	} else {
		if err != nil {
			return nil, err
		} else {
			for _, entry := range entries {
				if entry.IsDir() {
					container, _ := getRunningContainerInfoForId(entry.Name())
					if container.pid > 0 {
						containers = append(containers, container)
					}
				}
			}
			return containers, nil
		}
	}
}

func printRunningContainers() {
	containers, err := getRunningContainers()
	if err != nil {
		os.Exit(1)
	}

	fmt.Println("CONTAINER ID\tIMAGE\t\tCOMMAND")
	for _, container := range containers {
		fmt.Printf("%s\t%s\t%s\n", container.containerId, container.image, container.command)
	}
}
