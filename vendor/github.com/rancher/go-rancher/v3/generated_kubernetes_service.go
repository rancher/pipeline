package client

const (
	KUBERNETES_SERVICE_TYPE = "kubernetesService"
)

type KubernetesService struct {
	Resource `yaml:"-"`

	AccountId string `json:"accountId,omitempty" yaml:"account_id,omitempty"`

	AssignServiceIpAddress bool `json:"assignServiceIpAddress,omitempty" yaml:"assign_service_ip_address,omitempty"`

	BatchSize int64 `json:"batchSize,omitempty" yaml:"batch_size,omitempty"`

	ClusterId string `json:"clusterId,omitempty" yaml:"cluster_id,omitempty"`

	CompleteUpdate bool `json:"completeUpdate,omitempty" yaml:"complete_update,omitempty"`

	CreateIndex int64 `json:"createIndex,omitempty" yaml:"create_index,omitempty"`

	CreateOnly bool `json:"createOnly,omitempty" yaml:"create_only,omitempty"`

	Created string `json:"created,omitempty" yaml:"created,omitempty"`

	CurrentScale int64 `json:"currentScale,omitempty" yaml:"current_scale,omitempty"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	ExternalId string `json:"externalId,omitempty" yaml:"external_id,omitempty"`

	Fqdn string `json:"fqdn,omitempty" yaml:"fqdn,omitempty"`

	HealthState string `json:"healthState,omitempty" yaml:"health_state,omitempty"`

	InstanceIds []string `json:"instanceIds,omitempty" yaml:"instance_ids,omitempty"`

	IntervalMillis int64 `json:"intervalMillis,omitempty" yaml:"interval_millis,omitempty"`

	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`

	LbConfig *LbConfig `json:"lbConfig,omitempty" yaml:"lb_config,omitempty"`

	LbTargetConfig *LbTargetConfig `json:"lbTargetConfig,omitempty" yaml:"lb_target_config,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	PreviousRevisionId string `json:"previousRevisionId,omitempty" yaml:"previous_revision_id,omitempty"`

	PublicEndpoints []PublicEndpoint `json:"publicEndpoints,omitempty" yaml:"public_endpoints,omitempty"`

	RemoveTime string `json:"removeTime,omitempty" yaml:"remove_time,omitempty"`

	Removed string `json:"removed,omitempty" yaml:"removed,omitempty"`

	RevisionId string `json:"revisionId,omitempty" yaml:"revision_id,omitempty"`

	Selector string `json:"selector,omitempty" yaml:"selector,omitempty"`

	ServiceLinks []Link `json:"serviceLinks,omitempty" yaml:"service_links,omitempty"`

	StackId string `json:"stackId,omitempty" yaml:"stack_id,omitempty"`

	StartFirst bool `json:"startFirst,omitempty" yaml:"start_first,omitempty"`

	State string `json:"state,omitempty" yaml:"state,omitempty"`

	Transitioning string `json:"transitioning,omitempty" yaml:"transitioning,omitempty"`

	TransitioningMessage string `json:"transitioningMessage,omitempty" yaml:"transitioning_message,omitempty"`

	Upgrade *ServiceUpgrade `json:"upgrade,omitempty" yaml:"upgrade,omitempty"`

	Uuid string `json:"uuid,omitempty" yaml:"uuid,omitempty"`

	Vip string `json:"vip,omitempty" yaml:"vip,omitempty"`
}

type KubernetesServiceCollection struct {
	Collection
	Data   []KubernetesService `json:"data,omitempty"`
	client *KubernetesServiceClient
}

type KubernetesServiceClient struct {
	rancherClient *RancherClient
}

type KubernetesServiceOperations interface {
	List(opts *ListOpts) (*KubernetesServiceCollection, error)
	Create(opts *KubernetesService) (*KubernetesService, error)
	Update(existing *KubernetesService, updates interface{}) (*KubernetesService, error)
	ById(id string) (*KubernetesService, error)
	Delete(container *KubernetesService) error

	ActionActivate(*KubernetesService) (*Service, error)

	ActionCancelupgrade(*KubernetesService) (*Service, error)

	ActionCreate(*KubernetesService) (*Service, error)

	ActionDeactivate(*KubernetesService) (*Service, error)

	ActionError(*KubernetesService) (*Service, error)

	ActionFinishupgrade(*KubernetesService) (*Service, error)

	ActionGarbagecollect(*KubernetesService) (*Service, error)

	ActionPause(*KubernetesService) (*Service, error)

	ActionRemove(*KubernetesService) (*Service, error)

	ActionRestart(*KubernetesService) (*Service, error)

	ActionRollback(*KubernetesService, *ServiceRollback) (*Service, error)

	ActionUpdate(*KubernetesService) (*Service, error)

	ActionUpgrade(*KubernetesService, *ServiceUpgrade) (*Service, error)
}

func newKubernetesServiceClient(rancherClient *RancherClient) *KubernetesServiceClient {
	return &KubernetesServiceClient{
		rancherClient: rancherClient,
	}
}

func (c *KubernetesServiceClient) Create(container *KubernetesService) (*KubernetesService, error) {
	resp := &KubernetesService{}
	err := c.rancherClient.doCreate(KUBERNETES_SERVICE_TYPE, container, resp)
	return resp, err
}

func (c *KubernetesServiceClient) Update(existing *KubernetesService, updates interface{}) (*KubernetesService, error) {
	resp := &KubernetesService{}
	err := c.rancherClient.doUpdate(KUBERNETES_SERVICE_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *KubernetesServiceClient) List(opts *ListOpts) (*KubernetesServiceCollection, error) {
	resp := &KubernetesServiceCollection{}
	err := c.rancherClient.doList(KUBERNETES_SERVICE_TYPE, opts, resp)
	resp.client = c
	return resp, err
}

func (cc *KubernetesServiceCollection) Next() (*KubernetesServiceCollection, error) {
	if cc != nil && cc.Pagination != nil && cc.Pagination.Next != "" {
		resp := &KubernetesServiceCollection{}
		err := cc.client.rancherClient.doNext(cc.Pagination.Next, resp)
		resp.client = cc.client
		return resp, err
	}
	return nil, nil
}

func (c *KubernetesServiceClient) ById(id string) (*KubernetesService, error) {
	resp := &KubernetesService{}
	err := c.rancherClient.doById(KUBERNETES_SERVICE_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *KubernetesServiceClient) Delete(container *KubernetesService) error {
	return c.rancherClient.doResourceDelete(KUBERNETES_SERVICE_TYPE, &container.Resource)
}

func (c *KubernetesServiceClient) ActionActivate(resource *KubernetesService) (*Service, error) {

	resp := &Service{}

	err := c.rancherClient.doAction(KUBERNETES_SERVICE_TYPE, "activate", &resource.Resource, nil, resp)

	return resp, err
}

func (c *KubernetesServiceClient) ActionCancelupgrade(resource *KubernetesService) (*Service, error) {

	resp := &Service{}

	err := c.rancherClient.doAction(KUBERNETES_SERVICE_TYPE, "cancelupgrade", &resource.Resource, nil, resp)

	return resp, err
}

func (c *KubernetesServiceClient) ActionCreate(resource *KubernetesService) (*Service, error) {

	resp := &Service{}

	err := c.rancherClient.doAction(KUBERNETES_SERVICE_TYPE, "create", &resource.Resource, nil, resp)

	return resp, err
}

func (c *KubernetesServiceClient) ActionDeactivate(resource *KubernetesService) (*Service, error) {

	resp := &Service{}

	err := c.rancherClient.doAction(KUBERNETES_SERVICE_TYPE, "deactivate", &resource.Resource, nil, resp)

	return resp, err
}

func (c *KubernetesServiceClient) ActionError(resource *KubernetesService) (*Service, error) {

	resp := &Service{}

	err := c.rancherClient.doAction(KUBERNETES_SERVICE_TYPE, "error", &resource.Resource, nil, resp)

	return resp, err
}

func (c *KubernetesServiceClient) ActionFinishupgrade(resource *KubernetesService) (*Service, error) {

	resp := &Service{}

	err := c.rancherClient.doAction(KUBERNETES_SERVICE_TYPE, "finishupgrade", &resource.Resource, nil, resp)

	return resp, err
}

func (c *KubernetesServiceClient) ActionGarbagecollect(resource *KubernetesService) (*Service, error) {

	resp := &Service{}

	err := c.rancherClient.doAction(KUBERNETES_SERVICE_TYPE, "garbagecollect", &resource.Resource, nil, resp)

	return resp, err
}

func (c *KubernetesServiceClient) ActionPause(resource *KubernetesService) (*Service, error) {

	resp := &Service{}

	err := c.rancherClient.doAction(KUBERNETES_SERVICE_TYPE, "pause", &resource.Resource, nil, resp)

	return resp, err
}

func (c *KubernetesServiceClient) ActionRemove(resource *KubernetesService) (*Service, error) {

	resp := &Service{}

	err := c.rancherClient.doAction(KUBERNETES_SERVICE_TYPE, "remove", &resource.Resource, nil, resp)

	return resp, err
}

func (c *KubernetesServiceClient) ActionRestart(resource *KubernetesService) (*Service, error) {

	resp := &Service{}

	err := c.rancherClient.doAction(KUBERNETES_SERVICE_TYPE, "restart", &resource.Resource, nil, resp)

	return resp, err
}

func (c *KubernetesServiceClient) ActionRollback(resource *KubernetesService, input *ServiceRollback) (*Service, error) {

	resp := &Service{}

	err := c.rancherClient.doAction(KUBERNETES_SERVICE_TYPE, "rollback", &resource.Resource, input, resp)

	return resp, err
}

func (c *KubernetesServiceClient) ActionUpdate(resource *KubernetesService) (*Service, error) {

	resp := &Service{}

	err := c.rancherClient.doAction(KUBERNETES_SERVICE_TYPE, "update", &resource.Resource, nil, resp)

	return resp, err
}

func (c *KubernetesServiceClient) ActionUpgrade(resource *KubernetesService, input *ServiceUpgrade) (*Service, error) {

	resp := &Service{}

	err := c.rancherClient.doAction(KUBERNETES_SERVICE_TYPE, "upgrade", &resource.Resource, input, resp)

	return resp, err
}
