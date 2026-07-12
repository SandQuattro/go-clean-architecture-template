package app

import (
	"clean-arch-template/pkg/logger"
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
)

func PrintSystemData(log logger.Logger) {
	ctx := context.Background()

	log.Info(ctx, "====================== Application Info ==========================")

	log.Info(ctx, fmt.Sprintf("Environment: %s", os.Getenv("ENV_NAME")))

	// Hostname
	hostname, err := os.Hostname()
	if err != nil {
		log.Info(ctx, "Hostname: [Error getting hostname]")
	} else {
		log.Info(ctx, fmt.Sprintf("Hostname: %s", hostname))
	}

	log.Info(ctx, fmt.Sprintf("PID: %d", os.Getpid()))

	// IP Address
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Info(ctx, "IP Address: [Error getting IP address]")
	} else {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					log.Info(ctx, fmt.Sprintf("IP Address: %s", ipnet.IP.String()))
					break
				}
			}
		}
	}

	// CPU Cores
	log.Info(ctx, fmt.Sprintf("CPU Cores: %d", runtime.NumCPU()))

	// User
	user, err := os.UserHomeDir()
	if err != nil {
		log.Info(ctx, "User: [Error getting user home directory]")
	} else {
		log.Info(ctx, fmt.Sprintf("User: %s", user))
	}

	// Go Version
	log.Info(ctx, fmt.Sprintf("Go Version: %s", runtime.Version()))

	log.Info(ctx, "=================================================================")
}

func PrintMemoryInfo(log logger.Logger) {
	ctx := context.Background()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Info(ctx, "========================== Memory Info ==========================")
	log.Info(ctx, fmt.Sprintf("GOGC: %v", os.Getenv("GOGC")))
	log.Info(ctx, fmt.Sprintf("GOMEMLIMIT: %v", os.Getenv("GOMEMLIMIT")))
	log.Info(ctx, fmt.Sprintf("Allocated memory (Alloc): %v MB", bytesToMB(m.Alloc)))
	log.Info(ctx, fmt.Sprintf("Total allocated memory (TotalAlloc): %v MB", bytesToMB(m.TotalAlloc)))
	log.Info(ctx, fmt.Sprintf("Max memory available for application (Sys): %v MB", bytesToMB(m.Sys)))
	log.Info(ctx, fmt.Sprintf("Number of Goroutines: %d", runtime.NumGoroutine()))
	log.Info(ctx, "=================================================================")
}

func bytesToMB(b uint64) uint64 {
	return b / 1024 / 1024
}
