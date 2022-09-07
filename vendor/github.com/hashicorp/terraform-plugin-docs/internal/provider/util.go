package provider

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
)

func providerShortName(n string) string {
	return strings.TrimPrefix(n, "terraform-provider-")
}

func resourceShortName(name, providerName string) string {
	psn := providerShortName(providerName)
	return strings.TrimPrefix(name, psn+"_")
}

func copyFile(srcPath, dstPath string, mode os.FileMode) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// If the destination file already exists, we shouldn't blow it away
	dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}

func removeAllExt(file string) string {
	for {
		ext := filepath.Ext(file)
		if ext == "" || ext == file {
			return file
		}
		file = strings.TrimSuffix(file, ext)
	}
}

// resourceSchema determines whether there is a schema in the supplied schemas map which
// has either the providerShortName or the providerShortName concatenated with the
// templateFileName (stripped of file extension.
func resourceSchema(schemas map[string]*tfjson.Schema, providerShortName, templateFileName string) (*tfjson.Schema, string) {
	if schema, ok := schemas[providerShortName]; ok {
		return schema, providerShortName
	}

	resName := providerShortName + "_" + removeAllExt(templateFileName)

	if schema, ok := schemas[resName]; ok {
		return schema, resName
	}

	return nil, resName
}

func writeFile(path string, data string) error {
	dir, _ := filepath.Split(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("unable to make dir %q: %w", dir, err)
	}

	err = ioutil.WriteFile(path, []byte(data), 0644)
	if err != nil {
		return fmt.Errorf("unable to write file %q: %w", path, err)
	}

	return nil
}

func runCmd(cmd *exec.Cmd) ([]byte, error) {
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("error executing %q, %v", cmd.Path, cmd.Args)
		log.Printf(string(output))
		return nil, fmt.Errorf("error executing %q: %w", cmd.Path, err)
	}
	return output, nil
}

func cp(srcDir, dstDir string) error {
	err := filepath.Walk(srcDir, func(srcPath string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, srcPath)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dstDir, relPath)

		switch mode := f.Mode(); {
		case mode.IsDir():
			if err := os.Mkdir(dstPath, f.Mode()); err != nil && !os.IsExist(err) {
				return err
			}
		case mode.IsRegular():
			if err := copyFile(srcPath, dstPath, mode); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown file type (%d / %s) for %s", f.Mode(), f.Mode().String(), srcPath)
		}

		return nil
	})
	return err
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
