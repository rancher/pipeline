package pipeline

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/robfig/cron"
)

var ErrInvalidPipeline = errors.New("Invalid Pipeline definition")
var regName = regexp.MustCompile(`^[\w]+[\w-_]*`)

func Clean(p *Pipeline) {
	p.VersionSequence = ""
	p.RunCount = 0
	p.LastRunId = ""
	p.LastRunStatus = ""
	p.LastRunTime = 0
	p.NextRunTime = 0
	p.CommitInfo = ""
	p.Repository = ""
	p.Branch = ""
	p.TargetImage = ""
	p.File = ""
	p.Templates = nil
	p.WebHookId = 0
	p.WebHookToken = ""

	//set condition to nil if empty, for cleaner serialization
	for _, stage := range p.Stages {
		if stage.Conditions != nil && len(stage.Conditions.All) == 0 && len(stage.Conditions.Any) == 0 {
			stage.Conditions = nil
		}
		for _, step := range stage.Steps {
			if step.Conditions != nil && len(step.Conditions.All) == 0 && len(step.Conditions.Any) == 0 {
				step.Conditions = nil
			}
		}
	}

}

func Validate(p *Pipeline) error {
	if p.Name == "" {
		return errors.New("Pipeline name should not be null!")
	}

	//check scm step
	if len(p.Stages) < 1 || len(p.Stages[0].Steps) < 1 || p.Stages[0].Steps[0].Type != StepTypeSCM {
		return errors.New("SCM type should be the first step")
	}

	if err := checkCronSpec(p.CronTrigger.Spec); err != nil {
		return err
	}

	if err := checkStageName(p.Stages); err != nil {
		return err
	}

	if err := checkServiceName(p); err != nil {
		return err
	}

	for _, stage := range p.Stages {
		if err := checkCondition(stage.Conditions); err != nil {
			return err
		}
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
			return errors.Wrap(ErrInvalidPipeline, "repo field should not be null for SCM step")
		}
		if step.Branch == "" {
			return errors.Wrap(ErrInvalidPipeline, "repo field should not be null for SCM step")
		}
		if !strings.HasSuffix(step.Repository, ".git") {
			return errors.Wrap(ErrInvalidPipeline, "Invalid repo url for SCM step")
		}
	case StepTypeTask:
		if step.Image == "" {
			return errors.Wrap(ErrInvalidPipeline, "Image field should not be null for task step")
		}
	case StepTypeBuild:
		if step.TargetImage == "" {
			return errors.Wrap(ErrInvalidPipeline, "Target Image field should not be null for build step")
		}
	case StepTypeUpgradeService:
		if step.ImageTag == "" {
			return errors.Wrap(ErrInvalidPipeline, "Image field should not be null for upgradeService step")
		}
		if len(step.ServiceSelector) == 0 {
			return errors.Wrap(ErrInvalidPipeline, "Service selector should not be null for upgradeService step")
		}
	case StepTypeUpgradeStack:
		if step.StackName == "" {
			return errors.Wrap(ErrInvalidPipeline, "StackName should not be null for upgradeStack step")
		}
	case StepTypeUpgradeCatalog:
		if step.ExternalId == "" {
			return errors.Wrap(ErrInvalidPipeline, "ExternalId should not be null for upgradeCatalog step")
		}
	}
	if err := checkCondition(step.Conditions); err != nil {
		return err
	}
	return nil
}

func checkStageName(stages []*Stage) error {
	names := map[string]bool{}
	for _, stage := range stages {
		if stage.Name == "" {
			return errors.Wrap(ErrInvalidPipeline, "Stage name should not be null")
		}
		if _, ok := names[stage.Name]; ok {
			return errors.Wrapf(ErrInvalidPipeline, "Stage name '%v' duplicates", stage.Name)
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
	return errors.Wrapf(ErrInvalidPipeline, "parse cron expression got error:%v", err)
}

func checkCondition(conditions *PipelineConditions) error {
	if conditions == nil {
		return nil
	}
	for _, condition := range conditions.All {
		if !strings.Contains(condition, "=") {
			return errors.Wrapf(ErrInvalidPipeline, "condition '%s' is not valid, expected format 'xx=xx' or 'xx!=xx'", condition)
		}
	}
	for _, condition := range conditions.Any {
		if !strings.Contains(condition, "=") {
			return errors.Wrapf(ErrInvalidPipeline, "condition '%s' is not valid, expected format 'xx=xx' or 'xx!=xx'", condition)
		}
	}
	return nil
}

func checkServiceName(p *Pipeline) error {
	names := map[string]bool{}
	for _, stage := range p.Stages {
		for _, step := range stage.Steps {
			if step.IsService {
				if step.Alias == "" {
					return errors.Wrapf(ErrInvalidPipeline, "Please provide an alias when run as a service(in stage '%s')", stage.Name)
				}
				if _, ok := names[step.Alias]; ok {
					return errors.Wrapf(ErrInvalidPipeline, "Alias '%s' duplicates in as a service tasks", step.Alias)
				}
				names[step.Alias] = true
			}
		}
	}
	return nil
}

// IsValidName checks if name valid. limit to [a-zA-Z0-9-_]
func IsValidName(name string) error {
	match := regName.FindAllString(name, -1)
	if len(match) == 0 || (len(match[0]) != len(name)) {
		return fmt.Errorf("Invalid name %s, must contain [a-zA-Z0-9-_] characters only, cannot start with '-' or '_'", name)
	}
	return nil
}
