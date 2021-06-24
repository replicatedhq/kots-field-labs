package fieldlabs

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// this type is non-public in the Replicated CLI or I'd just import it
type kotsSingleSpec struct {
	Name     string   `json:"name"`
	Path     string   `json:"path"`
	Content  string   `json:"content"`
	Children []string `json:"children"`
}

// this function is non-public in the Replicated CLI or I'd just import it
func readYAMLDir(yamlDir string) (string, error) {

	var allKotsReleaseSpecs []kotsSingleSpec
	err := filepath.Walk(yamlDir, func(path string, info os.FileInfo, err error) error {
		spec, err := encodeKotsFile(yamlDir, path, info, err)
		if err != nil {
			return err
		} else if spec == nil {
			return nil
		}
		allKotsReleaseSpecs = append(allKotsReleaseSpecs, *spec)
		return nil
	})
	if err != nil {
		return "", errors.Wrapf(err, "walk %s", yamlDir)
	}

	jsonAllYamls, err := json.Marshal(allKotsReleaseSpecs)
	if err != nil {
		return "", errors.Wrap(err, "marshal spec")
	}
	return string(jsonAllYamls), nil
}

// this function is non-public in the Replicated CLI or I'd just import it
func encodeKotsFile(prefix, path string, info os.FileInfo, err error) (*kotsSingleSpec, error) {
	if err != nil {
		return nil, err
	}

	singlefile := strings.TrimPrefix(filepath.Clean(path), filepath.Clean(prefix)+"/")

	if info.IsDir() {
		return nil, nil
	}
	if strings.HasPrefix(info.Name(), ".") {
		return nil, nil
	}
	ext := filepath.Ext(info.Name())
	switch ext {
	case ".tgz", ".gz", ".yaml", ".yml":
		// continue
	default:
		return nil, nil
	}

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "read file %s", path)
	}

	var str string
	switch ext {
	case ".tgz", ".gz":
		str = base64.StdEncoding.EncodeToString(bytes)
	default:
		str = string(bytes)
	}

	return &kotsSingleSpec{
		Name:     info.Name(),
		Path:     singlefile,
		Content:  str,
		Children: []string{},
	}, nil
}
