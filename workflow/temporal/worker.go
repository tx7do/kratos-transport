package temporal

import (
	"context"
	"fmt"
	"sync"

	"go.temporal.io/sdk/worker"
)

// WorkflowWorker manages a Temporal worker that polls for tasks.
type WorkflowWorker struct {
	sync.RWMutex

	client  *WorkflowClient
	worker  worker.Worker
	opts    WorkerOptions
	running bool
	closed  bool
}

// NewWorker creates and returns a new WorkflowWorker.
// The worker is created but not started; call Start() to begin polling.
func (wc *WorkflowClient) NewWorker(opts WorkerOptions) (*WorkflowWorker, error) {
	if wc.client == nil {
		return nil, fmt.Errorf("temporal client is not connected")
	}

	w := worker.New(wc.client, opts.TaskQueue, opts.Options)

	// Register the default BrokerMessageWorkflow
	w.RegisterWorkflow(BrokerMessageWorkflow)

	// Register additional user-defined workflows
	for _, wf := range opts.Workflows {
		w.RegisterWorkflow(wf)
	}

	// Register activities
	for _, act := range opts.Activities {
		w.RegisterActivity(act)
	}

	return &WorkflowWorker{
		client: wc,
		worker: w,
		opts:   opts,
	}, nil
}

// Start starts the worker. This is a blocking call if the worker is configured
// to block, otherwise it starts in the background.
func (ww *WorkflowWorker) Start() error {
	ww.Lock()
	defer ww.Unlock()

	if ww.closed {
		return fmt.Errorf("worker is already stopped")
	}

	if ww.running {
		return fmt.Errorf("worker is already started")
	}

	if err := ww.worker.Start(); err != nil {
		return fmt.Errorf("failed to start temporal worker for task queue %s: %w", ww.opts.TaskQueue, err)
	}

	ww.running = true

	LogInfof("started temporal worker for task queue: %s", ww.opts.TaskQueue)

	return nil
}

// Stop gracefully stops the worker.
func (ww *WorkflowWorker) Stop() {
	ww.Lock()
	defer ww.Unlock()

	if ww.closed {
		return
	}

	if ww.worker != nil {
		ww.worker.Stop()
	}

	ww.closed = true

	LogInfof("stopped temporal worker for task queue: %s", ww.opts.TaskQueue)
}

// RegisterWorkflow registers a workflow function on the worker.
// Must be called before Start().
func (ww *WorkflowWorker) RegisterWorkflow(fn any) {
	ww.worker.RegisterWorkflow(fn)
}

// RegisterActivity registers an activity function or struct on the worker.
// Must be called before Start().
func (ww *WorkflowWorker) RegisterActivity(fn any) {
	ww.worker.RegisterActivity(fn)
}

// TaskQueue returns the task queue name this worker is listening to.
func (ww *WorkflowWorker) TaskQueue() string {
	return ww.opts.TaskQueue
}

// IsRunning returns whether the worker is still running.
func (ww *WorkflowWorker) IsRunning() bool {
	ww.RLock()
	defer ww.RUnlock()
	return ww.running && !ww.closed
}

// processMessageActivity is a built-in activity that wraps a handler function.
type processMessageActivity struct {
	handler func(ctx context.Context, body []byte) error
	client  *WorkflowClient
	topic   string
}

// ProcessMessage is the default activity implementation.
func (a *processMessageActivity) ProcessMessage(ctx context.Context, body []byte) error {
	ctx, span := a.client.startConsumerSpan(ctx, a.topic)
	defer func() {
		a.client.finishConsumerSpan(ctx, span, nil)
	}()

	return a.handler(ctx, body)
}

// StartSimpleWorker creates a worker with a single activity handler.
// This is the simplest way to start processing messages from a task queue.
func (wc *WorkflowClient) StartSimpleWorker(ctx context.Context, taskQueue string, handler func(ctx context.Context, body []byte) error, opts ...func(*WorkerOptions)) (*WorkflowWorker, error) {
	workerOpts := WorkerOptions{
		TaskQueue: taskQueue,
	}
	for _, o := range opts {
		o(&workerOpts)
	}

	ww, err := wc.NewWorker(workerOpts)
	if err != nil {
		return nil, err
	}

	// Register the message processing activity
	act := &processMessageActivity{
		handler: handler,
		client:  wc,
		topic:   taskQueue,
	}
	ww.RegisterActivity(act.ProcessMessage)

	if err := ww.Start(); err != nil {
		return nil, err
	}

	return ww, nil
}
