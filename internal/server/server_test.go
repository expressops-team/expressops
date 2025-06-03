package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"expressops/api/v1alpha1"
	pluginManager "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Variables y funciones no utilizadas están comentadas
// var mockGetPluginFunc func(name string) (pluginManager.Plugin, error)

// MockPlugin implements the Plugin interface for testing
type MockPlugin struct {
	mock.Mock
}

func (m *MockPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	args := m.Called(ctx, config, logger)
	return args.Error(0)
}

func (m *MockPlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	args := m.Called(ctx, request, shared)
	return args.Get(0), args.Error(1)
}

func (m *MockPlugin) FormatResult(result interface{}) (string, error) {
	args := m.Called(result)
	return args.String(0), args.Error(1)
}

// Registry of mocked plugins for testing - comentado por no usarse
// var mockPluginRegistry = make(map[string]*MockPlugin)

// Mock GetPlugin for testing - comentado por no usarse
// func mockGetPlugin(name string) (pluginManager.Plugin, error) {
//     if plugin, ok := mockPluginRegistry[name]; ok {
//         return plugin, nil
//     }
//     return nil, fmt.Errorf("plugin not found")
// }

func TestParseParams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
	}{
		{
			name:     "empty input",
			input:    "",
			expected: map[string]interface{}{},
		},
		{
			name:     "single param",
			input:    "key:value",
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:     "multiple params",
			input:    "key1:value1;key2:value2",
			expected: map[string]interface{}{"key1": "value1", "key2": "value2"},
		},
		{
			name:     "param with colon in value",
			input:    "key:value:with:colons",
			expected: map[string]interface{}{"key": "value:with:colons"},
		},
		{
			name:     "param with no value",
			input:    "key:",
			expected: map[string]interface{}{"key": ""},
		},
		{
			name:     "malformed param (no colon)",
			input:    "keyvalue",
			expected: map[string]interface{}{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseParams(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDynamicFlowHandler(t *testing.T) {
	// Setup
	logger := logrus.New()
	logger.SetOutput(io.Discard) // Suppress logs during tests
	timeout := 5 * time.Second

	// Reset flow registry before each test
	flowRegistry = map[string]v1alpha1.Flow{
		"test-flow": {
			Name: "test-flow",
			Pipeline: []v1alpha1.Step{
				{
					PluginRef: "test-plugin",
					Parameters: map[string]interface{}{
						"param1": "value1",
					},
				},
			},
		},
	}

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:           "valid flow request",
			url:            "/flow?flowName=test-flow",
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"flow":    "test-flow",
				"success": true,
				"count":   float64(1), // JSON converts integers to float64
			},
		},
		{
			name:           "missing flowName",
			url:            "/flow",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
		},
		{
			name:           "non-existent flow",
			url:            "/flow?flowName=non-existent",
			expectedStatus: http.StatusNotFound,
			expectedBody:   nil,
		},
		{
			name:           "flow with params",
			url:            "/flow?flowName=test-flow&params=extra:param",
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"flow":    "test-flow",
				"success": true,
				"count":   float64(1),
			},
		},
	}

	// Save the original function
	originalGetPlugin := pluginManager.GetPluginFunc

	// Create and configure a mock plugin
	mockPlugin := new(MockPlugin)
	mockPlugin.On("Execute", mock.Anything, mock.Anything, mock.Anything).Return("result", nil)
	mockPlugin.On("FormatResult", mock.Anything).Return("formatted result", nil)

	// Crear una función temporal con la misma firma que pluginManager.GetPlugin
	tempGetPlugin := func(name string) (pluginManager.Plugin, error) {
		if name == "test-plugin" {
			return mockPlugin, nil
		}
		return nil, fmt.Errorf("plugin not found")
	}

	// Reemplazar temporalmente la función
	pluginManager.GetPluginFunc = tempGetPlugin

	// Restore the original function after tests
	defer func() {
		pluginManager.GetPluginFunc = originalGetPlugin
	}()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.url, nil)
			w := httptest.NewRecorder()

			handler := dynamicFlowHandler(logger, timeout)
			handler(w, req)

			resp := w.Result()
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			if tc.expectedBody != nil {
				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err)

				var result map[string]interface{}
				err = json.Unmarshal(body, &result)
				assert.NoError(t, err)

				// Verify only the expected fields
				for k, v := range tc.expectedBody {
					assert.Equal(t, v, result[k])
				}
			}
		})
	}
}

func TestExecuteFlow(t *testing.T) {
	// Setup
	logger := logrus.New()
	logger.SetOutput(io.Discard) // Suppress logs during tests
	ctx := context.Background()

	// Simple flow with one step
	simpleFlow := v1alpha1.Flow{
		Name: "simple-flow",
		Pipeline: []v1alpha1.Step{
			{
				PluginRef: "simple-plugin",
				Parameters: map[string]interface{}{
					"param1": "value1",
				},
			},
		},
	}

	// Flow with multiple steps
	multiStepFlow := v1alpha1.Flow{
		Name: "multi-step-flow",
		Pipeline: []v1alpha1.Step{
			{
				PluginRef: "step1-plugin",
				Parameters: map[string]interface{}{
					"param1": "value1",
				},
			},
			{
				PluginRef: "step2-plugin",
				Parameters: map[string]interface{}{
					"param2": "value2",
				},
			},
		},
	}

	// Flow with a failing step
	failingFlow := v1alpha1.Flow{
		Name: "failing-flow",
		Pipeline: []v1alpha1.Step{
			{
				PluginRef: "failing-plugin",
				Parameters: map[string]interface{}{
					"param1": "value1",
				},
			},
		},
	}

	// Flow with a non-existent plugin
	nonExistentPluginFlow := v1alpha1.Flow{
		Name: "non-existent-plugin-flow",
		Pipeline: []v1alpha1.Step{
			{
				PluginRef: "non-existent-plugin",
			},
		},
	}

	// Create mocks for each plugin
	simplePlugin := new(MockPlugin)
	simplePlugin.On("Execute", mock.Anything, mock.Anything, mock.Anything).Return("simple result", nil)
	simplePlugin.On("FormatResult", mock.Anything).Return("formatted simple result", nil)

	step1Plugin := new(MockPlugin)
	step1Plugin.On("Execute", mock.Anything, mock.Anything, mock.Anything).Return("step1 result", nil)
	step1Plugin.On("FormatResult", mock.Anything).Return("formatted step1 result", nil)

	step2Plugin := new(MockPlugin)
	step2Plugin.On("Execute", mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			// Verify that the previous step result is available
			shared := args.Get(2).(*map[string]any)
			assert.Equal(t, "step1 result", (*shared)["previous_result"])
		}).
		Return("step2 result", nil)
	step2Plugin.On("FormatResult", mock.Anything).Return("formatted step2 result", nil)

	failingPlugin := new(MockPlugin)
	failingPlugin.On("Execute", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("plugin execution failed"))

	// Save the original function
	originalGetPlugin := pluginManager.GetPluginFunc

	// Replace GetPlugin with our mock version
	pluginManager.GetPluginFunc = func(name string) (pluginManager.Plugin, error) {
		switch name {
		case "simple-plugin":
			return simplePlugin, nil
		case "step1-plugin":
			return step1Plugin, nil
		case "step2-plugin":
			return step2Plugin, nil
		case "failing-plugin":
			return failingPlugin, nil
		default:
			return nil, fmt.Errorf("plugin not found")
		}
	}

	// Restore the original function after tests
	defer func() {
		pluginManager.GetPluginFunc = originalGetPlugin
	}()

	// Test cases
	tests := []struct {
		name             string
		flow             v1alpha1.Flow
		additionalParams map[string]interface{}
		expectedCount    int
		expectedError    bool
	}{
		{
			name:             "simple flow execution",
			flow:             simpleFlow,
			additionalParams: map[string]interface{}{},
			expectedCount:    1,
			expectedError:    false,
		},
		{
			name:             "multi-step flow execution",
			flow:             multiStepFlow,
			additionalParams: map[string]interface{}{},
			expectedCount:    2,
			expectedError:    false,
		},
		{
			name:             "flow with additional params",
			flow:             simpleFlow,
			additionalParams: map[string]interface{}{"extra": "param"},
			expectedCount:    1,
			expectedError:    false,
		},
		{
			name:             "flow with failing plugin",
			flow:             failingFlow,
			additionalParams: map[string]interface{}{},
			expectedCount:    1,
			expectedError:    true,
		},
		{
			name:             "flow with non-existent plugin",
			flow:             nonExistentPluginFlow,
			additionalParams: map[string]interface{}{},
			expectedCount:    1,
			expectedError:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			results := executeFlow(ctx, tc.flow, tc.additionalParams, req, logger, false)

			assert.Equal(t, tc.expectedCount, len(results))

			if tc.expectedError {
				// Verify that there is an error in the results
				hasError := false
				for _, res := range results {
					if result, ok := res.(map[string]interface{}); ok {
						if _, exists := result["error"]; exists {
							hasError = true
							break
						}
					}
				}
				assert.True(t, hasError, "An error was expected in the results")
			} else {
				// Verify that there are no errors in the results
				for _, res := range results {
					if result, ok := res.(map[string]interface{}); ok {
						_, hasError := result["error"]
						assert.False(t, hasError, "No errors were expected in the results")
					}
				}
			}
		})
	}
}

func TestExecuteFlowTimeout(t *testing.T) {
	// Setup
	logger := logrus.New()
	logger.SetOutput(io.Discard) // Suppress logs during tests

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Flow with a plugin that takes longer than the timeout
	slowFlow := v1alpha1.Flow{
		Name: "slow-flow",
		Pipeline: []v1alpha1.Step{
			{
				PluginRef: "slow-plugin",
			},
		},
	}

	// Create mock for the slow plugin
	slowPlugin := new(MockPlugin)
	slowPlugin.On("Execute", mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			// Simulate slow operation that should be interrupted by context timeout
			select {
			case <-time.After(500 * time.Millisecond):
				// This should never complete due to the context timeout
			case <-args.Get(0).(context.Context).Done():
				// This is the expected path - context should be cancelled
			}
		}).
		Return("", context.DeadlineExceeded) // Return the timeout error
	slowPlugin.On("FormatResult", "never reached").Return("", nil)
	slowPlugin.On("FormatResult", mock.Anything).Return("", nil)

	// Save the original function
	originalGetPlugin := pluginManager.GetPluginFunc

	// Replace GetPlugin with our mock version
	pluginManager.GetPluginFunc = func(name string) (pluginManager.Plugin, error) {
		if name == "slow-plugin" {
			return slowPlugin, nil
		}
		return nil, fmt.Errorf("plugin not found")
	}

	// Restore the original function after tests
	defer func() {
		pluginManager.GetPluginFunc = originalGetPlugin
	}()

	req := httptest.NewRequest("GET", "/test", nil)
	results := executeFlow(ctx, slowFlow, map[string]interface{}{}, req, logger, false)

	// Verify that there is an error due to timeout
	assert.Equal(t, 1, len(results))

	result, ok := results[0].(map[string]interface{})
	assert.True(t, ok, "The result should be a map")

	errorMsg, hasError := result["error"]
	assert.True(t, hasError, "There should be an error due to timeout")

	// Optionally, verify the error message contains timeout-related information
	errorStr, ok := errorMsg.(string)
	assert.True(t, ok, "The error should be a string")
	assert.Contains(t, errorStr, "context deadline exceeded", "Error should mention context deadline exceeded")
}
