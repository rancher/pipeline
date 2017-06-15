package pipeline

import (
	"io/ioutil"
	"os"
	"path"
	"strconv"

	"github.com/pkg/errors"

	"github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

const (
	Latest           = "latest"
	PipelineFileName = "pipeline.yml"
)

var ErrTemplatePathNotVaild = errors.New("TemplateBasePath is not a vaild directory path")
var ErrPipelineNotFound = errors.New("Pipeline Not found")

type PipelineContext struct {
	templateBase string
	provider     PipelineProvider
}

func (p *PipelineContext) GetPipelineByName(pipeline string) *Pipeline {
	return p.GetPipelineByNameAndVersion(pipeline, Latest)
}

func (p *PipelineContext) GetPipelineByNameAndVersion(pipeline, version string) *Pipeline {
	if version != Latest {
		return toPipeline(path.Join(p.templateBase, pipeline), version)
	}
	return getLatestVersionPipelineFile(path.Join(p.templateBase, pipeline))
}

func BuildPipelineContext(context *cli.Context, provider PipelineProvider) *PipelineContext {
	r := PipelineContext{
		templateBase: context.GlobalString("template_base_path"),
		provider:     provider,
	}
	f, err := os.Stat(r.templateBase)
	if err != nil {
		logrus.Fatal(err)
	}
	if !f.IsDir() {
		logrus.Fatal(ErrTemplatePathNotVaild)
	}
	return &r
}

func toPipeline(pipelineBasePath, version string) *Pipeline {
	targetPath := path.Join(pipelineBasePath, version, PipelineFileName)
	if _, err := os.Stat(targetPath); err == os.ErrNotExist {
		return nil
	}
	data, err := ioutil.ReadFile(targetPath)
	if err != nil {
		logrus.Error(err)
		return nil
	}
	r := Pipeline{}
	err = yaml.Unmarshal(data, &r)
	if err != nil {
		logrus.Error(err)
		return nil
	}
	r.File = string(data)
	r.VersionSequence = version
	return &r
}

func (p *PipelineContext) ListPipelines() []*Pipeline {
	var r []*Pipeline
	fi, _ := os.OpenFile(p.templateBase, os.O_RDONLY, 0755)
	defer fi.Close()
	pls, _ := fi.Readdirnames(0)
	for _, pl := range pls {
		targetPath := path.Join(p.templateBase, pl)
		if f, err := os.Stat(targetPath); err == nil {
			if f.IsDir() {
				r = append(r, getLatestVersionPipelineFile(targetPath))
			}
		}
	}
	return r
}

func getLatestVersionPipelineFile(pipelinePath string) *Pipeline {
	if _, er := os.Stat(pipelinePath); er == os.ErrNotExist {
		logrus.Error(errors.Wrapf(er, "pipeline <%s> not found", pipelinePath))
		return nil
	}
	fi, _ := os.OpenFile(pipelinePath, os.O_RDONLY, 0755)
	defer fi.Close()
	versions, _ := fi.Readdir(0)
	max := 0
	for _, vfi := range versions {
		if versionNum, err := strconv.Atoi(vfi.Name()); err == nil {
			if max <= versionNum {
				max = versionNum
			}
		}
	}
	return toPipeline(pipelinePath, strconv.Itoa(max))
}

func (p *PipelineContext) RunPipeline(pipeline string) (bool, error) {
	return p.RunPipelineWithVersion(pipeline, Latest)
}

func (p *PipelineContext) RunPipelineWithVersion(pipeline, version string) (bool, error) {
	pp := p.GetPipelineByNameAndVersion(pipeline, version)
	if pp == nil {
		return false, ErrPipelineNotFound
	}
	pp.RunPipeline(p.provider)
	return true, nil
}
