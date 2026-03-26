package confluence

import (
	"fmt"
	"regexp"
	"strings"
)

func ParseConfluenceURL(input string) (baseURL, pageID string, err error) {
	input = strings.TrimSpace(input)

	if matched, _ := regexp.MatchString(`^\d+$`, input); matched {
		return "", input, nil
	}

	re := regexp.MustCompile(`^(https?://[^\s/]+/wiki)/spaces/([A-Z0-9]+)/pages/(\d+)`)
	if matches := re.FindStringSubmatch(input); len(matches) == 4 {
		return matches[1], matches[3], nil
	}

	re2 := regexp.MustCompile(`^/spaces/([A-Z0-9]+)/pages/(\d+)`)
	if matches := re2.FindStringSubmatch(input); len(matches) == 3 {
		return "", matches[2], nil
	}

	re3 := regexp.MustCompile(`^(https?://[^\s/]+/wiki)/pages/(\d+)`)
	if matches := re3.FindStringSubmatch(input); len(matches) == 3 {
		return matches[1], matches[2], nil
	}

	return "", "", fmt.Errorf("could not parse Confluence URL: %s", input)
}
