package tests

import (
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

func loadEnv(fileName string) {
	data, err := os.ReadFile(fileName)
	if err != nil {
		log.Error().Msgf("Can'e load file %v, cause error %v", fileName, err)
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		item, _, _ := strings.Cut(line, "#")
		keyValue := strings.SplitN(strings.Trim(item, " \t"), "=", 2)
		if len((keyValue)) == 2 {
			os.Setenv(keyValue[0], keyValue[1])
		} else {
			os.Setenv(keyValue[0], "")
		}
	}
}

func init() {
	if _, err := os.Stat(".env"); err != nil {
		if _, err := os.Stat("../.env"); err != nil {
			if _, err := os.Stat("../../.env"); err != nil {
			} else {
				loadEnv("../../.env")
			}
		} else {
			loadEnv("../.env")
		}
	} else {
		loadEnv(".env")
	}
}
