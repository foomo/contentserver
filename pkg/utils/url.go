package utils

import (
	"net/url"
)

func IsValidURL(str string) bool {
	u, err := url.Parse(str)

	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	return err == nil && u.Scheme != "" && u.Host != ""
}
