package agent

type Agent struct {
	Name string
	ID   string
}

func New(name, id string) *Agent {
	return &Agent{
		Name: name,
		ID:   id,
	}
}

func (a *Agent) Start() error {
	return nil
}

func (a *Agent) Stop() error {
	return nil
}