package version

import (
	"clean-arch-template/pkg/logger"
	"context"
	_ "embed"
	"fmt"
)

var Version = "dev"

func PrintVersion(appName string, log logger.Logger) {
	log.Info(context.Background(), fmt.Sprintf("Application %s version %s", appName, Version))
}
