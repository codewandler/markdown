package competition

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// CollectSystemInfo gathers information about the current machine.
func CollectSystemInfo() SystemInfo {
	return SystemInfo{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		CPU:       detectCPU(),
		GoVersion: runtime.Version(),
		Hostname:  hostname(),
	}
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

func detectCPU() string {
	switch runtime.GOOS {
	case "linux":
		return cpuFromProc()
	case "darwin":
		return cpuFromSysctl()
	default:
		return runtime.GOARCH
	}
}

// cpuFromProc reads /proc/cpuinfo on Linux.
func cpuFromProc() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return runtime.GOARCH
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return runtime.GOARCH
}

// cpuFromSysctl reads CPU brand on macOS.
func cpuFromSysctl() string {
	out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output()
	if err != nil {
		return runtime.GOARCH
	}
	return strings.TrimSpace(string(out))
}
