package argo

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// WorkflowClient provides high-level access to Argo Workflows operations via REST API.
type WorkflowClient struct {
	mu      sync.RWMutex
	options ClientOptions
	client  *http.Client
	baseURL string
	running bool
}

// NewClient creates a new WorkflowClient and connects to the Argo Server.
func NewClient(opts ClientOptions) (*WorkflowClient, error) {
	if opts.ServerURL == "" {
		opts.ServerURL = defaultServerURL
	}
	if opts.Namespace == "" {
		opts.Namespace = defaultNamespace
	}

	baseURL := strings.TrimRight(opts.ServerURL, "/")

	httpClient := &http.Client{}
	if opts.InsecureSkipVerify {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	LogInfof("connected to Argo Server at %s (namespace: %s)", baseURL, opts.Namespace)

	return &WorkflowClient{
		options: opts,
		client:  httpClient,
		baseURL: baseURL,
		running: true,
	}, nil
}

// Close cleans up resources.
func (wc *WorkflowClient) Close() {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	wc.running = false
	wc.client.CloseIdleConnections()
	LogInfo("disconnected from Argo Server")
}

////////////////////////////////////////////////////////////////////////////////
/// HTTP Helpers
////////////////////////////////////////////////////////////////////////////////

func (wc *WorkflowClient) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body error: %w", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, wc.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("create request error: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if wc.options.Token != "" {
		req.Header.Set("Authorization", "Bearer "+wc.options.Token)
	}

	return req, nil
}

func (wc *WorkflowClient) doRequest(req *http.Request, result interface{}) error {
	resp, err := wc.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request error: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response error: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}

	if result != nil && len(data) > 0 {
		if err := json.Unmarshal(data, result); err != nil {
			return fmt.Errorf("unmarshal response error: %w", err)
		}
	}

	return nil
}

func (wc *WorkflowClient) namespace(optsNamespace string) string {
	if optsNamespace != "" {
		return optsNamespace
	}
	return wc.options.Namespace
}

////////////////////////////////////////////////////////////////////////////////
/// Workflow CRUD Operations
////////////////////////////////////////////////////////////////////////////////

// SubmitWorkflow submits a new workflow to Argo Workflows.
// The workflow definition is passed as a Workflow object.
func (wc *WorkflowClient) SubmitWorkflow(ctx context.Context, wf *Workflow, opts *SubmitOptions) (*Workflow, error) {
	ns := wc.options.Namespace
	if opts != nil && opts.Namespace != "" {
		ns = opts.Namespace
	}

	path := fmt.Sprintf("%s/%s", apiPathPrefix, ns)
	if opts != nil {
		params := url.Values{}
		if opts.ServerDryRun {
			params.Set("serverDryRun", "true")
		}
		for _, p := range opts.Parameters {
			params.Add("entrypointParameters", p)
		}
		if len(params) > 0 {
			path += "?" + params.Encode()
		}
	}

	req, err := wc.newRequest(ctx, http.MethodPost, path, wf)
	if err != nil {
		return nil, err
	}

	var result Workflow
	if err := wc.doRequest(req, &result); err != nil {
		return nil, fmt.Errorf("submit workflow error: %w", err)
	}

	LogInfof("submitted workflow %s in namespace %s", result.Metadata.Name, ns)
	return &result, nil
}

// GetWorkflow retrieves a workflow by name.
func (wc *WorkflowClient) GetWorkflow(ctx context.Context, name string, optsNamespace string) (*Workflow, error) {
	ns := wc.namespace(optsNamespace)
	path := fmt.Sprintf("%s/%s/%s", apiPathPrefix, ns, name)

	req, err := wc.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var result Workflow
	if err := wc.doRequest(req, &result); err != nil {
		return nil, fmt.Errorf("get workflow error: %w", err)
	}

	return &result, nil
}

// ListWorkflows lists workflows in a namespace.
func (wc *WorkflowClient) ListWorkflows(ctx context.Context, opts *ListOptions) (*WorkflowList, error) {
	ns := wc.options.Namespace
	if opts != nil && opts.Namespace != "" {
		ns = opts.Namespace
	}

	path := fmt.Sprintf("%s/%s", apiPathPrefix, ns)
	if opts != nil {
		params := url.Values{}
		if opts.LabelSelector != "" {
			params.Set("labelSelector", opts.LabelSelector)
		}
		if opts.FieldSelector != "" {
			params.Set("fieldSelector", opts.FieldSelector)
		}
		if opts.Limit > 0 {
			params.Set("limit", fmt.Sprintf("%d", opts.Limit))
		}
		if opts.Offset > 0 {
			params.Set("offset", fmt.Sprintf("%d", opts.Offset))
		}
		if len(params) > 0 {
			path += "?" + params.Encode()
		}
	}

	req, err := wc.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var result WorkflowList
	if err := wc.doRequest(req, &result); err != nil {
		return nil, fmt.Errorf("list workflows error: %w", err)
	}

	return &result, nil
}

// DeleteWorkflow deletes a workflow.
func (wc *WorkflowClient) DeleteWorkflow(ctx context.Context, name string, optsNamespace string) error {
	ns := wc.namespace(optsNamespace)
	path := fmt.Sprintf("%s/%s/%s", apiPathPrefix, ns, name)

	req, err := wc.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	if err := wc.doRequest(req, nil); err != nil {
		return fmt.Errorf("delete workflow error: %w", err)
	}

	LogInfof("deleted workflow %s in namespace %s", name, ns)
	return nil
}

////////////////////////////////////////////////////////////////////////////////
/// Workflow Lifecycle Operations
////////////////////////////////////////////////////////////////////////////////

// SuspendWorkflow suspends a running workflow.
func (wc *WorkflowClient) SuspendWorkflow(ctx context.Context, name string, optsNamespace string) error {
	ns := wc.namespace(optsNamespace)
	path := fmt.Sprintf("%s/%s/%s/suspend", apiPathPrefix, ns, name)

	req, err := wc.newRequest(ctx, http.MethodPut, path, nil)
	if err != nil {
		return err
	}

	if err := wc.doRequest(req, nil); err != nil {
		return fmt.Errorf("suspend workflow error: %w", err)
	}

	LogInfof("suspended workflow %s", name)
	return nil
}

// ResumeWorkflow resumes a suspended workflow.
func (wc *WorkflowClient) ResumeWorkflow(ctx context.Context, name string, optsNamespace string) error {
	ns := wc.namespace(optsNamespace)
	path := fmt.Sprintf("%s/%s/%s/resume", apiPathPrefix, ns, name)

	req, err := wc.newRequest(ctx, http.MethodPut, path, nil)
	if err != nil {
		return err
	}

	if err := wc.doRequest(req, nil); err != nil {
		return fmt.Errorf("resume workflow error: %w", err)
	}

	LogInfof("resumed workflow %s", name)
	return nil
}

// TerminateWorkflow terminates a running workflow.
func (wc *WorkflowClient) TerminateWorkflow(ctx context.Context, name string, optsNamespace string) error {
	ns := wc.namespace(optsNamespace)
	path := fmt.Sprintf("%s/%s/%s/terminate", apiPathPrefix, ns, name)

	req, err := wc.newRequest(ctx, http.MethodPut, path, nil)
	if err != nil {
		return err
	}

	if err := wc.doRequest(req, nil); err != nil {
		return fmt.Errorf("terminate workflow error: %w", err)
	}

	LogInfof("terminated workflow %s", name)
	return nil
}

// ResubmitWorkflow resubmits a workflow.
func (wc *WorkflowClient) ResubmitWorkflow(ctx context.Context, name string, optsNamespace string) (*Workflow, error) {
	ns := wc.namespace(optsNamespace)
	path := fmt.Sprintf("%s/%s/%s/resubmit", apiPathPrefix, ns, name)

	req, err := wc.newRequest(ctx, http.MethodPut, path, nil)
	if err != nil {
		return nil, err
	}

	var result Workflow
	if err := wc.doRequest(req, &result); err != nil {
		return nil, fmt.Errorf("resubmit workflow error: %w", err)
	}

	LogInfof("resubmitted workflow %s", name)
	return &result, nil
}

// RetryWorkflow retries a failed workflow.
func (wc *WorkflowClient) RetryWorkflow(ctx context.Context, name string, optsNamespace string) (*Workflow, error) {
	ns := wc.namespace(optsNamespace)
	path := fmt.Sprintf("%s/%s/%s/retry", apiPathPrefix, ns, name)

	req, err := wc.newRequest(ctx, http.MethodPut, path, nil)
	if err != nil {
		return nil, err
	}

	var result Workflow
	if err := wc.doRequest(req, &result); err != nil {
		return nil, fmt.Errorf("retry workflow error: %w", err)
	}

	LogInfof("retried workflow %s", name)
	return &result, nil
}

// StopWorkflow stops a workflow with an optional message.
func (wc *WorkflowClient) StopWorkflow(ctx context.Context, name string, optsNamespace string, message string) error {
	ns := wc.namespace(optsNamespace)
	path := fmt.Sprintf("%s/%s/%s/stop", apiPathPrefix, ns, name)

	body := map[string]string{}
	if message != "" {
		body["message"] = message
	}

	req, err := wc.newRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return err
	}

	if err := wc.doRequest(req, nil); err != nil {
		return fmt.Errorf("stop workflow error: %w", err)
	}

	LogInfof("stopped workflow %s", name)
	return nil
}

////////////////////////////////////////////////////////////////////////////////
/// Workflow Logs
////////////////////////////////////////////////////////////////////////////////

// GetWorkflowLogs retrieves logs for a workflow.
func (wc *WorkflowClient) GetWorkflowLogs(ctx context.Context, name string, optsNamespace string, podName string) (string, error) {
	ns := wc.namespace(optsNamespace)
	path := fmt.Sprintf("%s/%s/%s/log", apiPathPrefix, ns, name)
	if podName != "" {
		params := url.Values{}
		params.Set("podName", podName)
		path += "?" + params.Encode()
	}

	req, err := wc.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}

	resp, err := wc.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("get workflow logs error: %w", err)
	}
	defer resp.Body.Close()

	// 非 2xx 不能把错误响应体当日志返回
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("get workflow logs failed: HTTP %d: %s", resp.StatusCode, string(data))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read logs error: %w", err)
	}

	return string(data), nil
}
