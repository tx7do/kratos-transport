package pulsar

import (
	"regexp"
)

var re = regexp.MustCompile("^pulsar(\\+ssl)?://.*")

func hasUrlPrefix(url string) bool {
	return re.MatchString(url)
}

func refitUrl(url string, enableTLS bool) string {
	if !hasUrlPrefix(url) {
		prefix := "pulsar://"
		if enableTLS {
			prefix = "pulsar+ssl://"
		}
		url = prefix + url
	}
	return url
}
