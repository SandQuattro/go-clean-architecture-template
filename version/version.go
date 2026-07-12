package version

import (
	"clean-arch-template/config"
	"clean-arch-template/pkg/logger"
	"context"
	_ "embed"
	"fmt"
)

var Version = "dev"

func PrintVersion(cfg *config.Config, log logger.Logger) {
	log.Info(context.Background(), fmt.Sprintf("Application %s version %s", cfg.Name, Version))
}
