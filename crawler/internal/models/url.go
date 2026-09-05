package models

import (
	"strings"
)

type (
	SecondAndTopLevelDomain string
	Url                     string
)

func (url Url) GetSecondAndTopLevelDomain() SecondAndTopLevelDomain {
	/*

		>>> getDomain(https://www.youtube.com/watch?v=dQw4w9WgXcQ)
		https://www.youtube.com

	*/

	linkComponents := strings.Split(string(url), "://")
	// [0] == http:// || https://
	// [1] == www.example.com/thingy

	// protocol := linkComponents[0] + "://"
	domainWithoutRoutes, _, _ := strings.Cut(linkComponents[1], "/")

	domainLevels := strings.Split(domainWithoutRoutes, ".")

	topLevel := domainLevels[len(domainLevels)-1]
	secondLevel := domainLevels[len(domainLevels)-2]

	return SecondAndTopLevelDomain(secondLevel + "." + topLevel)
}

func (url Url) TrimTrailingSlash() Url {
	if string(url[len(url)-1]) != "/" {
		return url
	}

	return url[0 : len(url)-1]
}
