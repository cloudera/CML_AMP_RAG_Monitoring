package cbhttp

func NewTestingInstance(runner RunnerFunc) Client {
	return &Instance{doNoRetry: runner}
}
