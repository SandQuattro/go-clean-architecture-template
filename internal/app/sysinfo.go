package app

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"runtime"
)

func PrintSystemData() {
	slog.Info("====================== Application Info ==========================")

	slog.Info(fmt.Sprintf("Environment: %s", os.Getenv("ENV_NAME")))

	// Hostname
	hostname, err := os.Hostname()
	if err != nil {
		slog.Info("Hostname: [Error getting hostname]")
	} else {
		slog.Info(fmt.Sprintf("Hostname: %s", hostname))
	}

	slog.Info(fmt.Sprintf("PID: %d", os.Getpid()))

	// IP Address
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		slog.Info("IP Address: [Error getting IP address]")
	} else {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					slog.Info(fmt.Sprintf("IP Address: %s", ipnet.IP.String()))
					break
				}
			}
		}
	}

	// CPU Cores
	slog.Info(fmt.Sprintf("CPU Cores: %d", runtime.NumCPU()))

	// User
	user, err := os.UserHomeDir()
	if err != nil {
		slog.Info("User: [Error getting user home directory]")
	} else {
		slog.Info(fmt.Sprintf("User: %s", user))
	}

	// Go Version
	slog.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))

	// Go Home (GOROOT)
	slog.Info(fmt.Sprintf("Go Home (GOROOT): %s", runtime.GOROOT()))

	slog.Info("=================================================================")
}

func PrintMemoryInfo() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	slog.Info("========================== Memory Info ==========================")
	slog.Info(fmt.Sprintf("GOGC: %v", os.Getenv("GOGC")))
	slog.Info(fmt.Sprintf("GOMEMLIMIT: %v", os.Getenv("GOMEMLIMIT")))
	slog.Info(fmt.Sprintf("Allocated memory (Alloc): %v MB", bytesToMB(m.Alloc)))
	slog.Info(fmt.Sprintf("Total allocated memory (TotalAlloc): %v MB", bytesToMB(m.TotalAlloc)))
	slog.Info(fmt.Sprintf("Max memory available for application (Sys): %v MB", bytesToMB(m.Sys)))
	slog.Info(fmt.Sprintf("Number of Goroutines: %d", runtime.NumGoroutine()))
	slog.Info("=================================================================")
}

func bytesToMB(b uint64) uint64 {
	return b / 1024 / 1024
}
