package pipeline

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/pipeline/storer"
	"github.com/rancher/pipeline/util"
	"github.com/sluu99/uuid"

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
	storer       storer.Storer
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

func (p *PipelineContext) GetPipelineById(id string) *Pipeline {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return nil
	}
	filters := make(map[string]interface{})
	filters["key"] = id
	filters["kind"] = "pipeline"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		logrus.Errorf("Error %v filtering genericObjects by key", err)
		return nil
	}
	if len(goCollection.Data) == 0 {
		logrus.Errorf("Error %v filtering genericObjects by key", err)
		return nil
	}
	data := goCollection.Data[0]
	ppl := Pipeline{}
	json.Unmarshal([]byte(data.ResourceData["data"].(string)), &ppl)
	logrus.Infof("get pipeline:%v", ppl)
	return &ppl
}

func BuildPipelineContext(context *cli.Context, provider PipelineProvider) *PipelineContext {
	r := PipelineContext{
		templateBase: context.GlobalString("template_base_path"),
		provider:     provider,
		storer:       storer.InitLocalStorer(context.GlobalString("template_base_path")),
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

func (p *PipelineContext) CreatePipeline(pipeline Pipeline) error {
	pipeline.Id = uuid.Rand().Hex()
	b, err := json.Marshal(pipeline)
	if err != nil {
		return err
	}
	resourceData := map[string]interface{}{
		"data": string(b),
	}
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	_, err = apiClient.GenericObject.Create(&client.GenericObject{
		Name:         pipeline.Name,
		Key:          pipeline.Id,
		ResourceData: resourceData,
		Kind:         "pipeline",
	})
	logrus.Infof("created pipeline:%v", pipeline)

	return err
}
func (p *PipelineContext) UpdatePipeline(pipeline Pipeline) error {
	b, err := json.Marshal(pipeline)
	if err != nil {
		return err
	}
	resourceData := map[string]interface{}{
		"data": string(b),
	}
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	existing, err := apiClient.GenericObject.ById(pipeline.Id)
	logrus.Infof("existing pipeline:%v", existing)
	if err != nil {
		logrus.Errorf("find existing pipeline got error")
		return err
	}
	if existing != nil {
		existing, err = apiClient.GenericObject.Update(existing, &client.GenericObject{
			Name:         pipeline.Name,
			Key:          pipeline.Id,
			ResourceData: resourceData,
			Kind:         "pipeline",
		})
		if err != nil {
			return err
		}
	} else {
		return errors.New("cannot get existing pipeline to update")
	}
	return nil
}

func (p *PipelineContext) DeletePipeline(id string) error {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	filters := make(map[string]interface{})
	filters["key"] = id
	filters["kind"] = "pipeline"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		logrus.Errorf("Error %v filtering genericObjects by key", err)
		return err
	}
	if len(goCollection.Data) == 0 {
		return errors.New("cannot find pipeline to delete")
	}
	existing := goCollection.Data[0]

	err = apiClient.GenericObject.Delete(&existing)
	if err != nil {
		return err
	}

	return nil
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

//get all pipelines from GenericObject
func (p *PipelineContext) ListPipelines() []*Pipeline {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		logrus.Error("fail to get client")
		return nil
	}
	filters := make(map[string]interface{})
	filters["kind"] = "pipeline"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})

	if err != nil {
		logrus.Error("fail to list genericObject")
		return nil
	}
	var pipelines []*Pipeline
	for _, gobj := range goCollection.Data {
		b := []byte(gobj.ResourceData["data"].(string))
		a := &Pipeline{}
		json.Unmarshal(b, a)
		logrus.Infof("get pipeline:%v", a)
		pipelines = append(pipelines, a)
	}
	return pipelines
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

func (p *PipelineContext) RunPipeline(id string) (bool, error) {
	pp := p.GetPipelineById(id)
	if pp == nil {
		return false, ErrPipelineNotFound
	}
	pp.RunPipeline(p.provider)
	return true, nil
}

func (p *PipelineContext) RunPipelineWithVersion(pipeline, version string) (bool, error) {
	pp := p.GetPipelineByNameAndVersion(pipeline, version)
	if pp == nil {
		return false, ErrPipelineNotFound
	}
	pp.RunPipeline(p.provider)
	return true, nil
}
