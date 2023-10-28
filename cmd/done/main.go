package main

import (
	"log"
	"os"
	"sync"

	"github.com/cvhariharan/done/pkg/artifacts"
	"github.com/cvhariharan/done/pkg/models"
	"github.com/cvhariharan/done/pkg/runner"
	"gopkg.in/yaml.v3"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if len(os.Args) != 2 {
		log.Fatal("specify the job file")
	}

	contents, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	var jobFile models.JobFile
	err = yaml.Unmarshal(contents, &jobFile)
	if err != nil {
		log.Fatal(err)
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

	dockerArtifactManager := artifacts.NewDockerArtifactsManager()

	for _, v := range jobFile.Stages {
		var wg sync.WaitGroup
		for _, job := range stageMap[v] {
			wg.Add(1)
			go func(job models.Job) {
				runner.NewDockerRunner(job.Name, dockerArtifactManager).
					WithImage(job.Image).
					WithSrc(job.Src).
					WithCmd(job.Script).
					WithEnv(job.Variables).
					CreatesArtifacts(job.Artifacts).Run(os.Stdout)
				wg.Done()
			}(job)
		}
		wg.Wait()
	}
}
