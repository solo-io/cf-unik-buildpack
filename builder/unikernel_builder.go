package builder

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/cf-unik/cf-unik-buildpack/util"
	"github.com/cf-unik/pkg/errors"
	"github.com/cf-unik/unik/pkg/client"
)

const (
	//bases
	rump      = "rump"
	osv       = "osv"
	includeos = "includeos"

	//languages
	golang = "go"
	nodejs = "nodejs"
	c      = "c"
	cpp    = "cpp"
	java   = "java"
	python = "python"

	not_found = "not found"

	//providers
	virtualbox = "virtualbox"
	aws        = "aws"
	xen        = "xen"
	vsphere    = "vsphere"
)

/*needed metadata in manifest.yml:
PROVIDER=aws|virtualbox|vsphere
URL=10.10.10.10:3000
*/

func BuildUnikernel(sourcesDir, host string) error {
	sourceTar, err := ioutil.TempFile("", "sources.tar.gz.")
	if err != nil {
		logrus.WithError(err).Error("failed to create tmp tar file")
	}
	defer os.Remove(sourceTar.Name())
	if err := compress(sourcesDir, sourceTar.Name()); err != nil {
		return errors.New("failed to tar sources", err)
	}
	logrus.Infof("App packaged as tarball: %s\n", sourceTar.Name())

	provider := strings.ToLower(os.Getenv("PROVIDER"))
	switch provider {
	case virtualbox:
		fallthrough
	case aws:
		fallthrough
	case vsphere:
		logrus.Infof("using provider %s", provider)
	default:
		return errors.New("unsupported provider type "+provider, nil)
	}
	base, lang, err := detectLanguage(sourcesDir)
	if err != nil {
		return errors.New("detecting language", err)
	}

	imageName, err := util.GetAppName()
	if err != nil {
		return errors.New("could not get app name", err)
	}
	runArgs := os.Getenv("ARGS")
	logrus.WithFields(logrus.Fields{
		"sourcesDir": sourcesDir,
		"URL":        host,
		"imageName":  imageName,
		"base":       base,
		"lang":       lang,
		"provider":   provider,
		"runArgs":    runArgs,
	}).Infof("building unikernel")
	if err := doBuildRequest(sourceTar.Name(), host, imageName, base, lang, provider, runArgs); err != nil {
		return errors.New("failed building image. see UniK daemon logs for more information", err)
	}
	return nil
}

///http://blog.ralch.com/tutorial/golang-working-with-tar-and-gzip/
func compress(source, destination string) error {
	tarCmd := exec.Command("tar", "cf", destination, "-C", source, ".")
	if out, err := tarCmd.Output(); err != nil {
		return errors.New("running tar command: "+string(out), err)
	}
	return nil
}

//recurse directory, look for one of either:
func detectLanguage(sourcesDir string) (string, string, error) {
	var base, language string
	if err := filepath.Walk(sourcesDir, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(info.Name(), ".go") {
			language = golang
			base = rump
			return nil
		} else if strings.HasSuffix(info.Name(), ".py") {
			language = python
			base = rump
			return nil
		} else if strings.HasSuffix(info.Name(), ".java") || strings.HasSuffix(info.Name(), ".jar") || strings.HasSuffix(info.Name(), ".war") {
			language = java
			base = osv
			return nil
		} else if strings.HasSuffix(info.Name(), ".js") {
			//be careful we arent serving a website of some sort.. only use JS if we find no other kind of source files
			if language == "" {
				language = nodejs
				base = rump
			}
			return nil
		} else if strings.HasSuffix(info.Name(), ".c") {
			language = c
			base = rump
			return nil
		} else if strings.HasSuffix(info.Name(), ".cpp") {
			language = cpp
			base = includeos
			return nil
		}
		return nil
	}); err != nil {
		return "", "", errors.New("walking directory "+sourcesDir, err)
	}
	if language == "" {
		return "", "", errors.New("could not find language for project. should be Python, Go, Node, or Java", nil)
	}
	return base, language, nil
}

func doBuildRequest(sourcesTar, host, name, base, lang, provider, runArgs string) error {
	//TODO: add persistence/volume support
	var mountPoints []string
	image, err := client.UnikClient(host).Images().Build(name, sourcesTar, base, lang, provider, runArgs, mountPoints, true, false)
	if err != nil {
		return errors.New("building image failed", err)
	}
	logrus.Infof("successfully created image: %v", image)
	return nil
}
