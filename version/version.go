package version

import (
	_ "embed"
	"fmt"

	"clean-arch-template/config"
)

var Version = "dev"

func PrintVersion(cfg *config.Config) {
	println(fmt.Sprintf("%s version %s", cfg.App.Name, Version))
}
