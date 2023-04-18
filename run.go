package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

func createContainerID() string {
	randBytes := make([]byte, 6)
	rand.Read(randBytes)
	return fmt.Sprintf("%02x%02x%02x%02x%02x%02x",
		randBytes[0], randBytes[1], randBytes[2],
		randBytes[3], randBytes[4], randBytes[5])
}

func getContainerFSHome(containerID string) string {
	return getGockerContainersPath() + "/" + containerID + "/fs"
}

func createContainerDirectories(containerID string) {
	contHome := getGockerContainersPath() + "/" + containerID
	contDirs := []string{contHome + "/fs", contHome + "/fs/mnt", contHome + "/fs/upperdir", contHome + "/fs/workdir"}
	if err := createDirsIfDontExist(contDirs); err != nil {
		log.Fatalf("Unable to create required directories: %v\n", err)
	}
}

func mountOverlayFileSystem(containerID string, imageShaHex string) {
	var srcLayers []string
	pathManifest := getManifestPathForImage(imageShaHex)
	mani := manifest{}
	parseManifest(pathManifest, &mani)
	if len(mani) == 0 || len(mani[0].Layers) == 0 {
		log.Fatal("Could not find any layers.")
	}
	if len(mani) > 1 {
		log.Fatal("I don't know how to handle more than one manifest.")
	}

	imageBasePath := getBasePathForImage(imageShaHex)
	for _, layer := range mani[0].Layers {
		srcLayers = append([]string{imageBasePath + "/" + layer[:12] + "/fs"}, srcLayers...)
	}
	contFSHome := getContainerFSHome(containerID)
	mntOptions := "lowerdir=" + strings.Join(srcLayers, ":") + ",upperdir=" + contFSHome + "/upperdir,workdir=" + contFSHome + "/workdir"
	if err := unix.Mount("none", contFSHome+"/mnt", "overlay", 0, mntOptions); err != nil {
		log.Fatalf("Mount failed: %v\n", err)
	}
}

func unmountNetworkNamespace(containerID string) {
	netNsPath := getGockerNetNsPath() + "/" + containerID
	if err := unix.Unmount(netNsPath, 0); err != nil {
		log.Fatalf("Unable to mount network namespace: %v at %s", err, netNsPath)
	}
}

func unmountContainerFs(containerID string) {
	mountedPath := getGockerContainersPath() + "/" + containerID + "/fs/mnt"
	if err := unix.Unmount(mountedPath, 0); err != nil {
		log.Fatalf("Unable to mount container file system: %v at %s", err, mountedPath)
	}
}

func copyNameserverConfig(containerID string) error {
	resolvFilePaths := []string{
		"/var/run/systemd/resolve/resolv.conf",
		"/etc/gockerresolv.conf",
		"etc/resolv.conf",
	}
	for _, resolvFilePath := range resolvFilePaths {
		if _, err := os.Stat(resolvFilePath); os.IsNotExist(err) {
			continue
		} else {
			return copyFile(resolvFilePath, getContainerFSHome(containerID)+"/mnt/etc/resolv.conf")
		}
	}
	return nil
}

// // Called if this program is executed with "child-mode" as the first argument
// func execContainerCommand(mem int, swap int, pids int, cpus float64,
// 	containerID string, imageShaHex string, args []string) {
// 		mntPath := getContainerFSHome(containerID) + "/mnt"
// 		cmd := exec.Command(args[0], args[1:]...)
// 		cmd.Stdin = os.Stdin
// 		cmd.Stdout = os.Stdout
// 		cmd.Stderr = os.Stderr

// 		imgConfig := parseContainerConfig(imageShaHex)
// 		doOrDieWithMsg(unix.Sethostname([]byte(containerID)), "Unable to set hostname")
// 		doOrDieWithMsg(joinContainerNetworkNamespace(containerID), "Unable to join container network namespace")
		
// }
