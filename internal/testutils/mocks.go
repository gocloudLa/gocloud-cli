package testutils

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// MockSSMClient is a mock implementation of SSM client
type MockSSMClient struct {
	Parameters map[string]string
	Error      error
}

func (m *MockSSMClient) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	value, exists := m.Parameters[*params.Name]
	if !exists {
		return nil, &types.ParameterNotFound{}
	}

	return &ssm.GetParameterOutput{
		Parameter: &types.Parameter{
			Name:  params.Name,
			Value: aws.String(value),
		},
	}, nil
}

func (m *MockSSMClient) GetParametersByPath(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	var parameters []types.Parameter
	for name, value := range m.Parameters {
		if strings.HasPrefix(name, *params.Path) {
			parameters = append(parameters, types.Parameter{
				Name:  aws.String(name),
				Value: aws.String(value),
			})
		}
	}

	return &ssm.GetParametersByPathOutput{
		Parameters: parameters,
	}, nil
}

func (m *MockSSMClient) PutParameter(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	if m.Parameters == nil {
		m.Parameters = make(map[string]string)
	}
	m.Parameters[*params.Name] = *params.Value

	return &ssm.PutParameterOutput{}, nil
}

func (m *MockSSMClient) DeleteParameter(ctx context.Context, params *ssm.DeleteParameterInput, optFns ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	delete(m.Parameters, *params.Name)
	return &ssm.DeleteParameterOutput{}, nil
}

// MockFileSystem provides mock file system operations
type MockFileSystem struct {
	Files       map[string]string
	Directories map[string]bool
	Error       error
}

func (m *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	content, exists := m.Files[filename]
	if !exists {
		return nil, os.ErrNotExist
	}

	return []byte(content), nil
}

func (m *MockFileSystem) WriteFile(filename, content string) error {
	if m.Error != nil {
		return m.Error
	}

	if m.Files == nil {
		m.Files = make(map[string]string)
	}
	m.Files[filename] = content
	return nil
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if m.Error != nil {
		return m.Error
	}

	if m.Directories == nil {
		m.Directories = make(map[string]bool)
	}
	m.Directories[path] = true
	return nil
}

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	if _, exists := m.Files[name]; exists {
		return &MockFileInfo{name: name, isDir: false}, nil
	}

	if _, exists := m.Directories[name]; exists {
		return &MockFileInfo{name: name, isDir: true}, nil
	}

	return nil, os.ErrNotExist
}

// MockFileInfo implements os.FileInfo
type MockFileInfo struct {
	name  string
	isDir bool
}

func (m *MockFileInfo) Name() string       { return m.name }
func (m *MockFileInfo) Size() int64        { return 0 }
func (m *MockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *MockFileInfo) ModTime() time.Time { return time.Now() }
func (m *MockFileInfo) IsDir() bool        { return m.isDir }
func (m *MockFileInfo) Sys() interface{}   { return nil }

// MockCommandExecutor provides mock command execution
type MockCommandExecutor struct {
	Commands map[string]string
	Error    error
}

func (m *MockCommandExecutor) Command(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	if m.Error != nil {
		cmd = exec.Command("false") // This will always fail
	}
	return cmd
}

// MockUserInput provides mock user input
type MockUserInput struct {
	Responses []string
	Index     int
	Error     error
}

func (m *MockUserInput) ReadString(delim byte) (string, error) {
	if m.Error != nil {
		return "", m.Error
	}

	if m.Index >= len(m.Responses) {
		return "", m.Error
	}

	response := m.Responses[m.Index]
	m.Index++
	return response, nil
}

// CreateTempDir creates a temporary directory for testing
func CreateTempDir() (string, error) {
	return os.MkdirTemp("", "gocloud-test-*")
}

// CleanupTempDir removes a temporary directory
func CleanupTempDir(dir string) error {
	return os.RemoveAll(dir)
}
