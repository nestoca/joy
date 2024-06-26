// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package pr

import (
	"sync"
)

// Ensure, that PullRequestProviderMock does implement PullRequestProvider.
// If this is not the case, regenerate this file with moq.
var _ PullRequestProvider = &PullRequestProviderMock{}

// PullRequestProviderMock is a mock implementation of PullRequestProvider.
//
//	func TestSomethingThatUsesPullRequestProvider(t *testing.T) {
//
//		// make and configure a mocked PullRequestProvider
//		mockedPullRequestProvider := &PullRequestProviderMock{
//			CreateFunc: func(createParams CreateParams) (string, error) {
//				panic("mock out the Create method")
//			},
//			CreateInteractivelyFunc: func(branch string) error {
//				panic("mock out the CreateInteractively method")
//			},
//			EnsureInstalledAndAuthenticatedFunc: func() error {
//				panic("mock out the EnsureInstalledAndAuthenticated method")
//			},
//			ExistsFunc: func(branch string) (bool, error) {
//				panic("mock out the Exists method")
//			},
//			GetBranchesPromotingToEnvironmentFunc: func(env string) ([]string, error) {
//				panic("mock out the GetBranchesPromotingToEnvironment method")
//			},
//			GetPromotionEnvironmentFunc: func(branch string) (string, error) {
//				panic("mock out the GetPromotionEnvironment method")
//			},
//			SetPromotionEnvironmentFunc: func(branch string, env string) error {
//				panic("mock out the SetPromotionEnvironment method")
//			},
//		}
//
//		// use mockedPullRequestProvider in code that requires PullRequestProvider
//		// and then make assertions.
//
//	}
type PullRequestProviderMock struct {
	// CreateFunc mocks the Create method.
	CreateFunc func(createParams CreateParams) (string, error)

	// CreateInteractivelyFunc mocks the CreateInteractively method.
	CreateInteractivelyFunc func(branch string) error

	// EnsureInstalledAndAuthenticatedFunc mocks the EnsureInstalledAndAuthenticated method.
	EnsureInstalledAndAuthenticatedFunc func() error

	// ExistsFunc mocks the Exists method.
	ExistsFunc func(branch string) (bool, error)

	// GetBranchesPromotingToEnvironmentFunc mocks the GetBranchesPromotingToEnvironment method.
	GetBranchesPromotingToEnvironmentFunc func(env string) ([]string, error)

	// GetPromotionEnvironmentFunc mocks the GetPromotionEnvironment method.
	GetPromotionEnvironmentFunc func(branch string) (string, error)

	// SetPromotionEnvironmentFunc mocks the SetPromotionEnvironment method.
	SetPromotionEnvironmentFunc func(branch string, env string) error

	// calls tracks calls to the methods.
	calls struct {
		// Create holds details about calls to the Create method.
		Create []struct {
			// CreateParams is the createParams argument value.
			CreateParams CreateParams
		}
		// CreateInteractively holds details about calls to the CreateInteractively method.
		CreateInteractively []struct {
			// Branch is the branch argument value.
			Branch string
		}
		// EnsureInstalledAndAuthenticated holds details about calls to the EnsureInstalledAndAuthenticated method.
		EnsureInstalledAndAuthenticated []struct {
		}
		// Exists holds details about calls to the Exists method.
		Exists []struct {
			// Branch is the branch argument value.
			Branch string
		}
		// GetBranchesPromotingToEnvironment holds details about calls to the GetBranchesPromotingToEnvironment method.
		GetBranchesPromotingToEnvironment []struct {
			// Env is the env argument value.
			Env string
		}
		// GetPromotionEnvironment holds details about calls to the GetPromotionEnvironment method.
		GetPromotionEnvironment []struct {
			// Branch is the branch argument value.
			Branch string
		}
		// SetPromotionEnvironment holds details about calls to the SetPromotionEnvironment method.
		SetPromotionEnvironment []struct {
			// Branch is the branch argument value.
			Branch string
			// Env is the env argument value.
			Env string
		}
	}
	lockCreate                            sync.RWMutex
	lockCreateInteractively               sync.RWMutex
	lockEnsureInstalledAndAuthenticated   sync.RWMutex
	lockExists                            sync.RWMutex
	lockGetBranchesPromotingToEnvironment sync.RWMutex
	lockGetPromotionEnvironment           sync.RWMutex
	lockSetPromotionEnvironment           sync.RWMutex
}

// Create calls CreateFunc.
func (mock *PullRequestProviderMock) Create(createParams CreateParams) (string, error) {
	callInfo := struct {
		CreateParams CreateParams
	}{
		CreateParams: createParams,
	}
	mock.lockCreate.Lock()
	mock.calls.Create = append(mock.calls.Create, callInfo)
	mock.lockCreate.Unlock()
	if mock.CreateFunc == nil {
		var (
			sOut   string
			errOut error
		)
		return sOut, errOut
	}
	return mock.CreateFunc(createParams)
}

// CreateCalls gets all the calls that were made to Create.
// Check the length with:
//
//	len(mockedPullRequestProvider.CreateCalls())
func (mock *PullRequestProviderMock) CreateCalls() []struct {
	CreateParams CreateParams
} {
	var calls []struct {
		CreateParams CreateParams
	}
	mock.lockCreate.RLock()
	calls = mock.calls.Create
	mock.lockCreate.RUnlock()
	return calls
}

// CreateInteractively calls CreateInteractivelyFunc.
func (mock *PullRequestProviderMock) CreateInteractively(branch string) error {
	callInfo := struct {
		Branch string
	}{
		Branch: branch,
	}
	mock.lockCreateInteractively.Lock()
	mock.calls.CreateInteractively = append(mock.calls.CreateInteractively, callInfo)
	mock.lockCreateInteractively.Unlock()
	if mock.CreateInteractivelyFunc == nil {
		var (
			errOut error
		)
		return errOut
	}
	return mock.CreateInteractivelyFunc(branch)
}

// CreateInteractivelyCalls gets all the calls that were made to CreateInteractively.
// Check the length with:
//
//	len(mockedPullRequestProvider.CreateInteractivelyCalls())
func (mock *PullRequestProviderMock) CreateInteractivelyCalls() []struct {
	Branch string
} {
	var calls []struct {
		Branch string
	}
	mock.lockCreateInteractively.RLock()
	calls = mock.calls.CreateInteractively
	mock.lockCreateInteractively.RUnlock()
	return calls
}

// EnsureInstalledAndAuthenticated calls EnsureInstalledAndAuthenticatedFunc.
func (mock *PullRequestProviderMock) EnsureInstalledAndAuthenticated() error {
	callInfo := struct {
	}{}
	mock.lockEnsureInstalledAndAuthenticated.Lock()
	mock.calls.EnsureInstalledAndAuthenticated = append(mock.calls.EnsureInstalledAndAuthenticated, callInfo)
	mock.lockEnsureInstalledAndAuthenticated.Unlock()
	if mock.EnsureInstalledAndAuthenticatedFunc == nil {
		var (
			errOut error
		)
		return errOut
	}
	return mock.EnsureInstalledAndAuthenticatedFunc()
}

// EnsureInstalledAndAuthenticatedCalls gets all the calls that were made to EnsureInstalledAndAuthenticated.
// Check the length with:
//
//	len(mockedPullRequestProvider.EnsureInstalledAndAuthenticatedCalls())
func (mock *PullRequestProviderMock) EnsureInstalledAndAuthenticatedCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockEnsureInstalledAndAuthenticated.RLock()
	calls = mock.calls.EnsureInstalledAndAuthenticated
	mock.lockEnsureInstalledAndAuthenticated.RUnlock()
	return calls
}

// Exists calls ExistsFunc.
func (mock *PullRequestProviderMock) Exists(branch string) (bool, error) {
	callInfo := struct {
		Branch string
	}{
		Branch: branch,
	}
	mock.lockExists.Lock()
	mock.calls.Exists = append(mock.calls.Exists, callInfo)
	mock.lockExists.Unlock()
	if mock.ExistsFunc == nil {
		var (
			bOut   bool
			errOut error
		)
		return bOut, errOut
	}
	return mock.ExistsFunc(branch)
}

// ExistsCalls gets all the calls that were made to Exists.
// Check the length with:
//
//	len(mockedPullRequestProvider.ExistsCalls())
func (mock *PullRequestProviderMock) ExistsCalls() []struct {
	Branch string
} {
	var calls []struct {
		Branch string
	}
	mock.lockExists.RLock()
	calls = mock.calls.Exists
	mock.lockExists.RUnlock()
	return calls
}

// GetBranchesPromotingToEnvironment calls GetBranchesPromotingToEnvironmentFunc.
func (mock *PullRequestProviderMock) GetBranchesPromotingToEnvironment(env string) ([]string, error) {
	callInfo := struct {
		Env string
	}{
		Env: env,
	}
	mock.lockGetBranchesPromotingToEnvironment.Lock()
	mock.calls.GetBranchesPromotingToEnvironment = append(mock.calls.GetBranchesPromotingToEnvironment, callInfo)
	mock.lockGetBranchesPromotingToEnvironment.Unlock()
	if mock.GetBranchesPromotingToEnvironmentFunc == nil {
		var (
			stringsOut []string
			errOut     error
		)
		return stringsOut, errOut
	}
	return mock.GetBranchesPromotingToEnvironmentFunc(env)
}

// GetBranchesPromotingToEnvironmentCalls gets all the calls that were made to GetBranchesPromotingToEnvironment.
// Check the length with:
//
//	len(mockedPullRequestProvider.GetBranchesPromotingToEnvironmentCalls())
func (mock *PullRequestProviderMock) GetBranchesPromotingToEnvironmentCalls() []struct {
	Env string
} {
	var calls []struct {
		Env string
	}
	mock.lockGetBranchesPromotingToEnvironment.RLock()
	calls = mock.calls.GetBranchesPromotingToEnvironment
	mock.lockGetBranchesPromotingToEnvironment.RUnlock()
	return calls
}

// GetPromotionEnvironment calls GetPromotionEnvironmentFunc.
func (mock *PullRequestProviderMock) GetPromotionEnvironment(branch string) (string, error) {
	callInfo := struct {
		Branch string
	}{
		Branch: branch,
	}
	mock.lockGetPromotionEnvironment.Lock()
	mock.calls.GetPromotionEnvironment = append(mock.calls.GetPromotionEnvironment, callInfo)
	mock.lockGetPromotionEnvironment.Unlock()
	if mock.GetPromotionEnvironmentFunc == nil {
		var (
			sOut   string
			errOut error
		)
		return sOut, errOut
	}
	return mock.GetPromotionEnvironmentFunc(branch)
}

// GetPromotionEnvironmentCalls gets all the calls that were made to GetPromotionEnvironment.
// Check the length with:
//
//	len(mockedPullRequestProvider.GetPromotionEnvironmentCalls())
func (mock *PullRequestProviderMock) GetPromotionEnvironmentCalls() []struct {
	Branch string
} {
	var calls []struct {
		Branch string
	}
	mock.lockGetPromotionEnvironment.RLock()
	calls = mock.calls.GetPromotionEnvironment
	mock.lockGetPromotionEnvironment.RUnlock()
	return calls
}

// SetPromotionEnvironment calls SetPromotionEnvironmentFunc.
func (mock *PullRequestProviderMock) SetPromotionEnvironment(branch string, env string) error {
	callInfo := struct {
		Branch string
		Env    string
	}{
		Branch: branch,
		Env:    env,
	}
	mock.lockSetPromotionEnvironment.Lock()
	mock.calls.SetPromotionEnvironment = append(mock.calls.SetPromotionEnvironment, callInfo)
	mock.lockSetPromotionEnvironment.Unlock()
	if mock.SetPromotionEnvironmentFunc == nil {
		var (
			errOut error
		)
		return errOut
	}
	return mock.SetPromotionEnvironmentFunc(branch, env)
}

// SetPromotionEnvironmentCalls gets all the calls that were made to SetPromotionEnvironment.
// Check the length with:
//
//	len(mockedPullRequestProvider.SetPromotionEnvironmentCalls())
func (mock *PullRequestProviderMock) SetPromotionEnvironmentCalls() []struct {
	Branch string
	Env    string
} {
	var calls []struct {
		Branch string
		Env    string
	}
	mock.lockSetPromotionEnvironment.RLock()
	calls = mock.calls.SetPromotionEnvironment
	mock.lockSetPromotionEnvironment.RUnlock()
	return calls
}
