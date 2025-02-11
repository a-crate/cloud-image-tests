// Copyright 2024 Google LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hostnamevalidation

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-image-tests/utils"
)

const gcomment = "# Added by Google"

func testHostnameWindows(shortname string) error {
	command := "[System.Net.Dns]::GetHostName()"
	output, err := utils.RunPowershellCmd(command)
	if err != nil {
		return fmt.Errorf("Error getting hostname: %v", err)
	}
	hostname := strings.TrimSpace(output.Stdout)

	if hostname != shortname {
		return fmt.Errorf("Expected Hostname: '%s', Actual Hostname: '%s'", shortname, hostname)
	}
	return nil
}

func testHostnameLinux(shortname string) error {
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("couldn't determine local hostname")
	}

	if hostname != shortname {
		return fmt.Errorf("hostname does not match metadata. Expected: %q got: %q", shortname, hostname)
	}

	// If hostname is FQDN then lots of tools (e.g. ssh-keygen) have issues
	if strings.Contains(hostname, ".") {
		return fmt.Errorf("hostname contains '.'")
	}
	return nil
}

func TestHostname(t *testing.T) {
	metadataHostname, err := utils.GetMetadata(utils.Context(t), "instance", "hostname")
	if err != nil {
		t.Fatalf(" still couldn't determine metadata hostname")
	}

	// 'hostname' in metadata is fully qualified domain name.
	shortname := strings.Split(metadataHostname, ".")[0]

	if runtime.GOOS == "windows" {
		if err = testHostnameWindows(shortname); err != nil {
			t.Fatalf("windows hostname error: %v", err)
		}
	} else {
		if err = testHostnameLinux(shortname); err != nil {
			t.Fatalf("linux hostname error: %v", err)
		}
	}
}

// TestCustomHostname tests the 'fully qualified domain name'.
func TestCustomHostname(t *testing.T) {
	image, err := utils.GetMetadata(utils.Context(t), "instance", "image")
	if err != nil {
		t.Fatalf("Couldn't get image from metadata")
	}

	// SLES doesn't support custom hostnames yet.
	if strings.Contains(image, "sles") {
		t.Skip("SLES doesn't support custom hostnames.")
	}
	if strings.Contains(image, "suse") {
		t.Skip("SUSE doesn't support custom hostnames.")
	}

	// Ubuntu doesn't support custom hostnames yet.
	if strings.Contains(image, "ubuntu") {
		t.Skip("Ubuntu doesn't support custom hostnames.")
	}

	TestFQDN(t)
}

// TestFQDN tests the 'fully qualified domain name'.
func TestFQDN(t *testing.T) {
	utils.LinuxOnly(t)
	ctx := utils.Context(t)
	// TODO Zonal DNS is breaking this test case in EL9.
	image, err := utils.GetMetadata(ctx, "instance", "image")
	if err != nil {
		t.Fatalf("Couldn't get image from metadata")
	}
	if strings.Contains(image, "almalinux-9") {
		// Zonal DNS change is breaking EL9.
		t.Skip("Broken on EL9")
	}
	if strings.Contains(image, "centos-stream-9") {
		// Zonal DNS change is breaking EL9.
		t.Skip("Broken on EL9")
	}
	if strings.Contains(image, "rhel-9") {
		// Zonal DNS change is breaking EL9.
		t.Skip("Broken on EL9")
	}
	if strings.Contains(image, "rocky-linux-9") {
		// Zonal DNS change is breaking EL9.
		t.Skip("Broken on EL9")
	}

	metadataHostname, err := utils.GetMetadata(ctx, "instance", "hostname")
	if err != nil {
		t.Fatalf("couldn't determine metadata hostname")
	}

	// Get the hostname with FQDN.
	cmd := exec.Command("/bin/hostname", "-f")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("hostname command failed")
	}
	hostname := strings.TrimRight(string(out), " \n")

	if hostname != metadataHostname {
		t.Errorf("hostname does not match metadata. Expected: %q got: %q", metadataHostname, hostname)
	}
}

func md5Sum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("couldn't open file: %v", err)
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

type sshKeyHash struct {
	file os.FileInfo
	hash string
}

// TestHostKeysGeneratedOnces checks that the guest agent only generates host keys one time.
func TestHostKeysGeneratedOnce(t *testing.T) {
	utils.LinuxOnly(t)
	sshDir := "/etc/ssh/"
	sshfiles, err := ioutil.ReadDir(sshDir)
	if err != nil {
		t.Fatalf("Couldn't read files from ssh dir")
	}

	var hashes []sshKeyHash
	for _, file := range sshfiles {
		if !strings.HasSuffix(file.Name(), "_key.pub") {
			continue
		}
		hash, err := md5Sum(sshDir + file.Name())
		if err != nil {
			t.Fatalf("Couldn't hash file: %v", err)
		}
		hashes = append(hashes, sshKeyHash{file, hash})
	}

	image, err := utils.GetMetadata(utils.Context(t), "instance", "image")
	if err != nil {
		t.Fatalf("Couldn't get image from metadata")
	}

	var restart string
	switch {
	case strings.Contains(image, "rhel-6"), strings.Contains(image, "centos-6"):
		restart = "initctl"
	default:
		restart = "systemctl"
	}

	cmd := exec.Command(restart, "restart", "google-guest-agent")
	err = cmd.Run()
	if err != nil {
		t.Errorf("Failed to restart guest agent: %v", err)
	}

	sshfiles, err = ioutil.ReadDir(sshDir)
	if err != nil {
		t.Fatalf("Couldn't read files from ssh dir")
	}

	var hashesAfter []sshKeyHash
	for _, file := range sshfiles {
		if !strings.HasSuffix(file.Name(), "_key.pub") {
			continue
		}
		hash, err := md5Sum(sshDir + file.Name())
		if err != nil {
			t.Fatalf("Couldn't hash file: %v", err)
		}
		hashesAfter = append(hashesAfter, sshKeyHash{file, hash})
	}

	if len(hashes) != len(hashesAfter) {
		t.Fatalf("Hashes changed after restarting guest agent")
	}

	for i := 0; i < len(hashes); i++ {
		if hashes[i].file.Name() != hashesAfter[i].file.Name() || hashes[i].hash != hashesAfter[i].hash {
			t.Fatalf("Hashes changed after restarting guest agent")
		}
	}
}

func TestHostsFile(t *testing.T) {
	utils.LinuxOnly(t)
	ctx := utils.Context(t)
	image, err := utils.GetMetadata(ctx, "instance", "image")
	if err != nil {
		t.Fatalf("couldn't get image from metadata")
	}
	if strings.Contains(image, "sles") {
		// SLES does not have dhclient or the dhclient exit hook.
		t.Skip("Not supported on SLES")
	}
	if strings.Contains(image, "suse") {
		// SLES does not have dhclient or the dhclient exit hook.
		t.Skip("Not supported on SUSE")
	}
	if strings.Contains(image, "ubuntu") {
		// Ubuntu does not have dhclient or the dhclient exit hook.
		t.Skip("Not supported on Ubuntu")
	}
	if strings.Contains(image, "almalinux-9") {
		// Does not have dhclient or the dhclient exit hook.
		t.Skip("Not supported on EL9")
	}
	if strings.Contains(image, "centos-stream-9") {
		// Does not have dhclient or the dhclient exit hook.
		t.Skip("Not supported on EL9")
	}
	if strings.Contains(image, "rhel-9") {
		// Does not have dhclient or the dhclient exit hook.
		t.Skip("Not supported on EL9")
	}
	if strings.Contains(image, "rocky-linux-9") {
		// Does not have dhclient or the dhclient exit hook.
		t.Skip("Not supported on EL9")
	}
	if strings.Contains(image, "debian-12") {
		// Does not have dhclient or the dhclient exit hook.
		t.Skip("Not supported on Debian 12")
	}

	b, err := ioutil.ReadFile("/etc/hosts")
	if err != nil {
		t.Fatalf("Couldn't read /etc/hosts")
	}
	ip, err := utils.GetMetadata(ctx, "instance", "network-interfaces", "0", "ip")
	if err != nil {
		t.Fatalf("Couldn't get ip from metadata")
	}
	hostname, err := utils.GetMetadata(ctx, "instance", "hostname")
	if err != nil {
		t.Fatalf("Couldn't get hostname from metadata")
	}
	targetLineHost := fmt.Sprintf("%s %s %s  %s\n", ip, hostname, strings.Split(hostname, ".")[0], gcomment)
	targetLineMetadata := fmt.Sprintf("%s %s  %s\n", "169.254.169.254", "metadata.google.internal", gcomment)
	if !strings.Contains(string(b), targetLineHost) {
		t.Fatalf("/etc/hosts does not contain host record.")
	}
	if !strings.Contains(string(b), targetLineMetadata) {
		t.Fatalf("/etc/hosts does not contain metadata server record.")
	}
}
