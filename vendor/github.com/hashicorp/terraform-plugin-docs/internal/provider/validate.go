package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/cli"
)

func Validate(ui cli.Ui) error {
	dirExists := func(name string) bool {
		if _, err := os.Stat(name); err != nil {
			return false
		}

		return true
	}

	switch {
	default:
		ui.Warn("no website detected, exiting")
	case dirExists("templates"):
		ui.Info("detected templates directory, running checks...")
		err := validateTemplates(ui, "templates")
		if err != nil {
			return err
		}
		if dirExists("examples") {
			ui.Info("detected examples directory for templates, running checks...")
			err = validateExamples(ui, "examples")
			if err != nil {
				return err
			}
		}
		return err
	case dirExists("docs"):
		ui.Info("detected static docs directory, running checks")
		return validateStaticDocs(ui, "docs")
	case dirExists("website"):
		ui.Info("detected legacy website directory, running checks")
		return validateLegacyWebsite(ui, "website")
	}

	return nil
}

func validateExamples(ui cli.Ui, dir string) error {
	return nil
}

func validateTemplates(ui cli.Ui, dir string) error {
	checks := []check{
		checkAllowedFiles(
			"index.md",
			"index.md.tmpl",
		),
		checkAllowedDirs(
			"data-sources",
			"guides",
			"resources",
		),
		checkBlockedExtensions(
			".html.md.tmpl",
		),
		checkAllowedExtensions(
			".md",
			".md.tmpl",
		),
	}
	issues := []issue{}
	for _, c := range checks {
		checkIssues, err := c(dir)
		if err != nil {
			return err
		}
		issues = append(issues, checkIssues...)
	}
	for _, issue := range issues {
		ui.Warn(fmt.Sprintf("%s: %s", issue.file, issue.message))
	}
	if len(issues) > 0 {
		return fmt.Errorf("invalid templates directory")
	}
	return nil
}

func validateStaticDocs(ui cli.Ui, dir string) error {
	checks := []check{
		checkAllowedFiles(
			"index.md",
		),
		checkAllowedDirs(
			"data-sources",
			"guides",
			"resources",
		),
		checkBlockedExtensions(
			".html.md.tmpl",
			".html.md",
			".md.tmpl",
		),
		checkAllowedExtensions(
			".md",
		),
	}
	issues := []issue{}
	for _, c := range checks {
		checkIssues, err := c(dir)
		if err != nil {
			return err
		}
		issues = append(issues, checkIssues...)
	}
	for _, issue := range issues {
		ui.Warn(fmt.Sprintf("%s: %s", issue.file, issue.message))
	}
	if len(issues) > 0 {
		return fmt.Errorf("invalid templates directory")
	}
	return nil
}

func validateLegacyWebsite(ui cli.Ui, dir string) error {
	panic("not implemented")
}

type issue struct {
	file    string
	message string
}

type check func(dir string) ([]issue, error)

func checkBlockedExtensions(exts ...string) check {
	return func(dir string) ([]issue, error) {
		issues := []issue{}
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			for _, ext := range exts {
				if strings.HasSuffix(path, ext) {
					_, file := filepath.Split(path)
					issues = append(issues, issue{
						file:    path,
						message: fmt.Sprintf("the extension for %q is not supported", file),
					})
					break
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		return issues, nil
	}
}

func checkAllowedExtensions(exts ...string) check {
	return func(dir string) ([]issue, error) {
		issues := []issue{}
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			valid := false
			for _, ext := range exts {
				if strings.HasSuffix(path, ext) {
					valid = true
					break
				}
			}
			if !valid {
				_, file := filepath.Split(path)
				issues = append(issues, issue{
					file:    path,
					message: fmt.Sprintf("the extension for %q is not expected", file),
				})
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		return issues, nil
	}
}

func checkAllowedDirs(dirs ...string) check {
	allowedDirs := map[string]bool{}
	for _, d := range dirs {
		allowedDirs[d] = true
	}

	return func(dir string) ([]issue, error) {
		issues := []issue{}

		f, err := os.Open(dir)
		if err != nil {
			return nil, err
		}
		infos, err := f.Readdir(-1)
		if err != nil {
			return nil, err
		}

		for _, fi := range infos {
			if !fi.IsDir() {
				continue
			}

			if !allowedDirs[fi.Name()] {
				issues = append(issues, issue{
					file:    filepath.Join(dir, fi.Name()),
					message: fmt.Sprintf("directory %q is not allowed", fi.Name()),
				})
			}
		}

		return issues, nil
	}
}

func checkAllowedFiles(dirs ...string) check {
	allowedFiles := map[string]bool{}
	for _, d := range dirs {
		allowedFiles[d] = true
	}

	return func(dir string) ([]issue, error) {
		issues := []issue{}

		f, err := os.Open(dir)
		if err != nil {
			return nil, err
		}
		infos, err := f.Readdir(-1)
		if err != nil {
			return nil, err
		}

		for _, fi := range infos {
			if fi.IsDir() {
				continue
			}

			if !allowedFiles[fi.Name()] {
				issues = append(issues, issue{
					file:    filepath.Join(dir, fi.Name()),
					message: fmt.Sprintf("file %q is not allowed", fi.Name()),
				})
			}
		}

		return issues, nil
	}
}
