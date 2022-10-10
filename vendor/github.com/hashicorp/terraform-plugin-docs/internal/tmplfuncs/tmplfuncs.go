package tmplfuncs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func PrefixLines(prefix, text string) string {
	return prefix + strings.Join(strings.Split(text, "\n"), "\n"+prefix)
}

func CodeFile(format, file string) (string, error) {
	// paths are relative to the rendering process work dir, which
	// may be undesirable, probably need to think about it
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(wd, file)
	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("unable to read content from %q: %w", file, err)
	}

	sContent := strings.TrimSpace(string(content))
	if sContent == "" {
		return "", fmt.Errorf("no file content in %q", file)
	}

	md := &strings.Builder{}
	_, err = md.WriteString("```" + format + "\n")
	if err != nil {
		return "", err
	}

	_, err = md.WriteString(sContent)
	if err != nil {
		return "", err
	}

	_, err = md.WriteString("\n```")
	if err != nil {
		return "", err
	}

	return md.String(), nil
}
