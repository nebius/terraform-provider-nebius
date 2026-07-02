package version

import (
	"errors"

	"github.com/blang/semver/v4"
)

const versionString = "0.6.23"

func BuildVersion() (string, error) {
	_, err := semver.Parse(versionString)
	if err != nil {
		return "unknown", errors.New("unknown version")
	}
	return versionString, nil
}
