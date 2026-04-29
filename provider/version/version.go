package version

import (
	"errors"

	"github.com/blang/semver/v4"
)

const versionString = "0.6.4"

func BuildVersion() (string, error) {
	_, err := semver.Parse(versionString)
	if err != nil {
		return "unknown", errors.New("Unknown version")
	}
	return versionString, nil
}
