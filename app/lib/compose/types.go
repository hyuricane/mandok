package compose

import (
	"bytes"
	"strings"
)

type RepoAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Registry string `json:"registry"`
}

type ComposeError struct {
	s []string
}

func (err *ComposeError) Error() string {
	if err == nil {
		return ""
	}
	return strings.Join(err.s, "\n")
}

func NewComposeError(buffErr *bytes.Buffer) error {
	str := buffErr.String()
	if str == "" {
		return nil
	}
	strs := strings.Split(str, "\n")
	cleanstrs := []string{}
	for _, s := range strs {
		if s == "" {
			continue
		}
		if strings.Contains(s, "level=warning") {
			continue
		}
		if s == "no configuration file provided: not found" {
			continue
		}
		cleanstrs = append(cleanstrs, s)
	}
	if len(cleanstrs) > 0 {
		return &ComposeError{s: cleanstrs}
	}
	return nil
}
