package version

import (
	_ "embed"
	"fmt"
	"log/slog"

	"clean-arch-template/config"
)

var Version = "dev"

func PrintVersion(cfg *config.Config) {
	slog.Info(fmt.Sprintf("Application %s version %s", cfg.App.Name, Version))
}
