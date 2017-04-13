package main

import (
	"flag"
	"github.com/cf-unik/cf-unik-buildpack/builder"
	"github.com/Sirupsen/logrus"
	"os"
	"github.com/cf-unik/cf-unik-buildpack/runner"
)

func main() {
	build := flag.Bool("build", false, "run in staging mode (compile unikernel)")
	run := flag.Bool("run", false, "run in runner mode (proxy for unikernel)")
	buildDir := flag.String("build-dir", "", "directory containing sources")
	flag.Parse()
	host := os.Getenv("URL")
	if host == "" {
		logrus.Fatal("must provide UniK URL with URL env var")
	}
	if *build {
		logrus.Info("building unikernel")
		if *buildDir == "" {
			logrus.Fatal("-build-dir must be set")
		}
		if err := builder.BuildUnikernel(*buildDir, host); err != nil {
			logrus.Fatal("failed building unikernel", err)
		}
	} else if *run {
		logrus.Info("running unikernel")
		if err := runner.RunUnikernel(host); err != nil {
			logrus.Fatal("failed running unikernel", err)
		}
	} else {
		logrus.Fatal("must provide either -build or -run flag")
	}
}