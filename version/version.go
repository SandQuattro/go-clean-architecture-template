package version

import (
	"clean-arch-template/config"
	_ "embed"
	"fmt"
	"log/slog"
)

var Version = "dev"

func PrintVersion(cfg *config.Config) {
	slog.Info(fmt.Sprintf("Application %s version %s", cfg.Name, Version))
}
