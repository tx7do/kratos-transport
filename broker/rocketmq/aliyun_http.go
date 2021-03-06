package rocketmq

import (
	"bytes"
	"encoding/gob"
	aliyun "github.com/aliyunmq/mq-http-go-sdk"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/gogap/errors"
	"github.com/tx7do/kratos-transport/broker"
	"strings"
	"sync"
	"time"
)

type aliyunBroker struct {
	nameServers   []string
	nameServerUrl string

	accessKey     string
	secretKey     string
	securityToken string

	instanceName string
	groupName    string
	retryCount   int
	namespace    string

	log *log.Helper

	connected bool
	sync.RWMutex
	opts broker.Options

	client    aliyun.MQClient
	producers map[string]aliyun.MQProducer
}

func newAliyunHttpBroker(options broker.Options) broker.Broker {
	return &aliyunBroker{
		producers:  make(map[string]aliyun.MQProducer),
		opts:       options,
		log:        log.NewHelper(log.GetLogger()),
		retryCount: 2,
	}
}

func (r *aliyunBroker) Name() string {
	return "rocketmq_http"
}

func (r *aliyunBroker) Address() string {
	if len(r.nameServers) > 0 {
		return r.nameServers[0]
	} else if r.nameServerUrl != "" {
		return r.nameServerUrl
	}
	return defaultAddr
}

func (r *aliyunBroker) Options() broker.Options {
	return r.opts
}

func (r *aliyunBroker) Init(opts ...broker.Option) error {
	r.opts.Apply(opts...)

	if v, ok := r.opts.Context.Value(nameServersKey{}).([]string); ok {
		r.nameServers = v
	}
	if v, ok := r.opts.Context.Value(nameServerUrlKey{}).(string); ok {
		r.nameServerUrl = v
	}
	if v, ok := r.opts.Context.Value(accessKey{}).(string); ok {
		r.accessKey = v
	}
	if v, ok := r.opts.Context.Value(secretKey{}).(string); ok {
		r.secretKey = v
	}
	if v, ok := r.opts.Context.Value(securityTokenKey{}).(string); ok {
		r.securityToken = v
	}
	if v, ok := r.opts.Context.Value(retryCountKey{}).(int); ok {
		r.retryCount = v
	}
	if v, ok := r.opts.Context.Value(namespaceKey{}).(string); ok {
		r.namespace = v
	}
	if v, ok := r.opts.Context.Value(instanceNameKey{}).(string); ok {
		r.instanceName = v
	}
	if v, ok := r.opts.Context.Value(groupNameKey{}).(string); ok {
		r.groupName = v
	}

	return nil
}

func (r *aliyunBroker) Connect() error {
	r.RLock()
	if r.connected {
		r.RUnlock()
		return nil
	}
	r.RUnlock()

	endpoint := r.Address()
	client := aliyun.NewAliyunMQClient(endpoint, r.accessKey, r.secretKey, r.securityToken)
	r.client = client

	r.Lock()
	r.connected = true
	r.Unlock()

	return nil
}

func (r *aliyunBroker) Disconnect() error {
	r.RLock()
	if !r.connected {
		r.RUnlock()
		return nil
	}
	r.RUnlock()

	r.Lock()
	defer r.Unlock()

	r.client = nil

	r.connected = false
	return nil
}

func (r *aliyunBroker) Publish(topic string, msg broker.Any, opts ...broker.PublishOption) error {
	if r.opts.Codec != nil {
		var err error
		buf, err := r.opts.Codec.Marshal(msg)
		if err != nil {
			return err
		}
		return r.publish(topic, buf, opts...)
	} else {
		switch t := msg.(type) {
		case []byte:
			return r.publish(topic, t, opts...)
		case string:
			return r.publish(topic, []byte(t), opts...)
		default:
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			if err := enc.Encode(msg); err != nil {
				return err
			}
			return r.publish(topic, buf.Bytes(), opts...)
		}
	}
}

func (r *aliyunBroker) publish(topic string, msg []byte, opts ...broker.PublishOption) error {
	options := broker.PublishOptions{}
	for _, o := range opts {
		o(&options)
	}

	if r.client == nil {
		return errors.New("client is nil")
	}

	r.Lock()
	p, ok := r.producers[topic]
	if !ok {
		p = r.client.GetProducer(r.instanceName, topic)
		if p == nil {
			r.Unlock()
			return errors.New("create producer failed")
		}

		r.producers[topic] = p
	} else {
	}
	r.Unlock()

	aMsg := aliyun.PublishMessageRequest{
		MessageBody: string(msg),
	}

	if v, ok := options.Context.Value(headerKey{}).(map[string]string); ok {
		aMsg.Properties = v
	}

	_, err := p.PublishMessage(aMsg)
	if err != nil {
		r.log.Errorf("[rocketmq]: send message error: %s\n", err)
		return err
	}

	return nil
}

func (r *aliyunBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	if r.client == nil {
		return nil, errors.New("client is nil")
	}

	options := broker.SubscribeOptions{
		AutoAck: true,
		Queue:   r.groupName,
	}
	for _, o := range opts {
		o(&options)
	}

	mqConsumer := r.client.GetConsumer(r.instanceName, topic, options.Queue, "")

	sub := &aliyunSubscriber{
		opts:    options,
		topic:   topic,
		handler: handler,
		binder:  binder,
		reader:  mqConsumer,
		done:    make(chan struct{}),
	}

	go r.doConsume(sub)

	return sub, nil
}

func (r *aliyunBroker) doConsume(sub *aliyunSubscriber) {
	for {
		endChan := make(chan int)
		respChan := make(chan aliyun.ConsumeMessageResponse)
		errChan := make(chan error)
		go func() {
			select {
			case resp := <-respChan:
				{
					var err error
					var m broker.Message
					for _, msg := range resp.Messages {

						p := &aliyunPublication{
							topic:  msg.Message,
							reader: sub.reader,
							m:      &m,
							rm:     []string{msg.ReceiptHandle},
							ctx:    r.opts.Context,
						}

						m.Headers = msg.Properties

						if sub.binder != nil {
							m.Body = sub.binder()
						}

						if r.opts.Codec != nil {
							if err := r.opts.Codec.Unmarshal([]byte(msg.MessageBody), m.Body); err != nil {
								p.err = err
							}
						} else {
							m.Body = []byte(msg.MessageBody)
						}

						err = sub.handler(sub.opts.Context, p)
						if err != nil {
							r.log.Errorf("[rocketmq]: process message failed: %v", err)
						}

						if sub.opts.AutoAck {
							if err = p.Ack(); err != nil {
								// ?????????????????????????????????????????????????????????????????????????????????
								if errAckItems, ok := err.(errors.ErrCode).Context()["Detail"].([]aliyun.ErrAckItem); ok {
									for _, errAckItem := range errAckItems {
										r.log.Errorf("ErrorHandle:%s, ErrorCode:%s, ErrorMsg:%s\n",
											errAckItem.ErrorHandle, errAckItem.ErrorCode, errAckItem.ErrorMsg)
									}
								} else {
									r.log.Error("ack err =", err)
								}
								time.Sleep(time.Duration(3) * time.Second)
							}
						}
					}

					endChan <- 1
				}
			case err := <-errChan:
				{
					// Topic???????????????????????????
					if strings.Contains(err.(errors.ErrCode).Error(), "MessageNotExist") {
						//r.log.Debug("No new message, continue!")
					} else {
						r.log.Error(err)
						time.Sleep(time.Duration(3) * time.Second)
					}
					endChan <- 1
				}
			case <-time.After(35 * time.Second):
				{
					//r.log.Debug("Timeout of consumer message ??")
					endChan <- 1
				}

			case sub.done <- struct{}{}:
				return
			}
		}()

		// ???????????????????????????????????????????????????35s???
		// ?????????????????????Topic??????????????????????????????????????????????????????3s???3s??????????????????????????????????????????????????????
		sub.reader.ConsumeMessage(respChan, errChan,
			3, // ??????????????????3????????????????????????16?????????
			3, // ???????????????3s?????????????????????30s??????
		)
		<-endChan
	}
}
