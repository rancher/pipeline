package pipeline

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/robfig/cron"
)

func Validate(p *Pipeline) error {
	if p.Name == "" {
		return errors.New("Pipeline name should not be null!")
	}

	//check scm step
	if len(p.Stages) < 1 || len(p.Stages[0].Steps) < 1 || p.Stages[0].Steps[0].Type != StepTypeSCM {
		return errors.New("SCM type should be the first step")
	}

	if err := checkCronSpec(p.TriggerSpec); err != nil {
		return err
	}

	if err := checkStageName(p.Stages); err != nil {
		return err
	}

	for _, stage := range p.Stages {
		for _, step := range stage.Steps {
			if err := validateStep(step); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateStep(step *Step) error {
	switch step.Type {
	case StepTypeSCM:
		if step.Repository == "" {
			return errors.New("repo field should not be null for SCM step")
		}
		if step.Branch == "" {
			return errors.New("repo field should not be null for SCM step")
		}
		if !strings.HasSuffix(step.Repository, ".git") {
			return errors.New("Invalid repo url for SCM step")
		}
	case StepTypeTask:
		if step.Image == "" {
			return errors.New("Image field should not be null for task step")
		}
	case StepTypeBuild:
		if step.TargetImage == "" {
			return errors.New("Target Image field should not be null for build step")
		}
	case StepTypeUpgradeService:
		if step.Tag == "" {
			return errors.New("Image field should not be null for upgradeService step")
		}
		if len(step.ServiceSelector) == 0 {
			return errors.New("Service selector should not be null for upgradeService step")
		}
	case StepTypeUpgradeStack:
		if step.StackName == "" {
			return errors.New("StackName should not be null for upgradeStack step")
		}
	case StepTypeUpgradeCatalog:
		//TODO
	}
	return nil
}

func checkStageName(stages []*Stage) error {
	names := map[string]bool{}
	for _, stage := range stages {
		if stage.Name == "" {
			return errors.New("Stage name should not be null")
		}
		if _, ok := names[stage.Name]; ok {
			return errors.New(fmt.Sprintf("Stage name '%v' duplicates", stage.Name))
		}
		names[stage.Name] = true
	}
	return nil
}

func checkCronSpec(spec string) error {
	if spec == "" {
		return nil
	}
	_, err := cron.ParseStandard(spec)
	return err
}
