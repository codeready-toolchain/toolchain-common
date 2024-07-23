package memberoperatorconfig

import (
    "github.com/stretchr/testify/mock"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/client-go/rest"
)

type MockConfigGetter struct {
    mock.Mock
}

func (m *MockConfigGetter) GetCRTConfiguration(config *rest.Config, scheme *runtime.Scheme) (Configuration, error) {
    args := m.Called(config, scheme)
    return args.Get(0).(Configuration), args.Error(1)
}
