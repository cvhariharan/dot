package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/cvhariharan/done/pkg/artifacts"
	"github.com/cvhariharan/done/pkg/models"
	"github.com/cvhariharan/done/pkg/runner"
	"github.com/go-playground/validator/v10"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

var validate *validator.Validate

func main() {
	ctx := context.Background()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var jobFilePath string
	flag.StringVar(&jobFilePath, "f", "done.yml", "Job File Path")

	var mountDockerSocket bool
	flag.BoolVar(&mountDockerSocket, "m", false, "Mount docker socket")
	flag.Parse()

	contents, err := os.ReadFile(jobFilePath)
	if err != nil {
		log.Fatal(err)
	}

	var jobFile models.JobFile
	err = yaml.Unmarshal(contents, &jobFile)
	if err != nil {
		log.Fatal(err)
	}

	validate = validator.New(validator.WithRequiredStructEnabled())
	err = validate.Struct(jobFile)
	if err != nil {
		log.Fatalf("Err(s):\n%+v\n", err)
	}

	stageMap := make(map[models.Stage][]models.Job)
	for _, v := range jobFile.Stages {
		stageMap[v] = make([]models.Job, 0)
	}

	for _, v := range jobFile.Jobs {
		if _, ok := stageMap[v.Stage]; !ok {
			log.Fatalf("stage not defined: %s", v.Stage)
		}
		stageMap[v.Stage] = append(stageMap[v.Stage], v)
	}

	dockerArtifactManager := artifacts.NewDockerArtifactsManager(".artifacts")

	for _, v := range jobFile.Stages {
		var eg errgroup.Group
		for _, job := range stageMap[v] {
			jobCtx, cancel := context.WithTimeout(ctx, time.Hour)
			defer cancel()

			func(job models.Job) {
				eg.Go(func() error {
					return runner.NewDockerRunner(job.Name, dockerArtifactManager, runner.DockerRunnerOptions{ShowImagePull: true, Stdout: os.Stdout, Stderr: os.Stderr, MountDockerSocket: mountDockerSocket}).
						WithImage(job.Image).
						WithSrc(job.Src).
						WithCmd(job.Script).
						WithEnv(job.Variables).
						CreatesArtifacts(job.Artifacts).Run(jobCtx)
				})
			}(job)
		}
		err := eg.Wait()
		if err != nil {
			log.Fatal(err)
		}
	}
}
