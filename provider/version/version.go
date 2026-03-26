package version

import (
	"errors"

	"github.com/blang/semver/v4"
)

var versionString string

func BuildVersion() (string, error) {
	_, err := semver.Parse(versionString)
	if err != nil {
		return "unknown", errors.New("Unknown version")
	}
	return versionString, nil
}
