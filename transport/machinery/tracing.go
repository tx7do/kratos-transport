package machinery

import (
	"context"
	"encoding/json"
	"github.com/RichardKnop/machinery/v2/tasks"
	"github.com/tx7do/kratos-transport/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	SignatureNameKey              = attribute.Key("signature.name")
	SignatureUUIDKey              = attribute.Key("signature.uuid")
	SignatureGroupUUIDKey         = attribute.Key("signature.group.uuid")
	SignatureChordCallbackUUIDKey = attribute.Key("signature.chord.callback.uuid")
	SignatureChordCallbackNameKey = attribute.Key("signature.chord.callback.name")
	ChainTasksLengthKey           = attribute.Key("chain.tasks.length")
	GroupUUIDKey                  = attribute.Key("group.uuid")
	GroupTasksLengthKey           = attribute.Key("group.tasks.length")
	GroupConcurrencyKey           = attribute.Key("group.concurrency")
	GroupTasksKey                 = attribute.Key("group.tasks")
	ChordCallbackUUIDKey          = attribute.Key("chord.callback.uuid")
)

func headersWithSpan(tracer *tracing.Tracer, ctx context.Context, headers *tasks.Headers) {
	carrier := NewMessageCarrier(headers)
	tracer.Inject(ctx, carrier)
}

func annotateSpanWithSignatureInfo(span trace.Span, signature *tasks.Signature) {
	attrs := []attribute.KeyValue{
		SignatureNameKey.String(signature.Name),
		SignatureUUIDKey.String(signature.UUID),
	}
	if signature.GroupUUID != "" {
		attrs = append(attrs, SignatureGroupUUIDKey.String(signature.GroupUUID))
	}

	if signature.ChordCallback != nil {
		attrs = append(attrs, SignatureChordCallbackUUIDKey.String(signature.ChordCallback.UUID))
		attrs = append(attrs, SignatureChordCallbackNameKey.String(signature.ChordCallback.Name))
	}

	span.SetAttributes(attrs...)
}

func annotateSpanWithGroupInfo(tracer *tracing.Tracer, ctx context.Context, span trace.Span, group *tasks.Group, sendConcurrency int) {
	attrs := []attribute.KeyValue{
		GroupUUIDKey.String(group.GroupUUID),
		GroupTasksLengthKey.Int(len(group.Tasks)),
		GroupConcurrencyKey.Int(sendConcurrency),
	}
	if taskUUIDs, err := json.Marshal(group.GetUUIDs()); err == nil {
		attrs = append(attrs, GroupTasksKey.String(string(taskUUIDs)))
	} else {
		attrs = append(attrs, GroupTasksKey.StringSlice(group.GetUUIDs()))
	}

	span.SetAttributes(attrs...)

	for _, signature := range group.Tasks {
		headersWithSpan(tracer, ctx, &signature.Headers)
	}
}

func annotateSpanWithChainInfo(tracer *tracing.Tracer, ctx context.Context, span trace.Span, chain *tasks.Chain) {
	attrs := []attribute.KeyValue{
		ChainTasksLengthKey.Int(len(chain.Tasks)),
	}
	span.SetAttributes(attrs...)

	for _, signature := range chain.Tasks {
		headersWithSpan(tracer, ctx, &signature.Headers)
	}
}

func annotateSpanWithChordInfo(tracer *tracing.Tracer, ctx context.Context, span trace.Span, chord *tasks.Chord, sendConcurrency int) {

	attrs := []attribute.KeyValue{
		ChordCallbackUUIDKey.String(chord.Callback.UUID),
	}
	span.SetAttributes(attrs...)

	headersWithSpan(tracer, ctx, &chord.Callback.Headers)

	annotateSpanWithGroupInfo(tracer, ctx, span, chord.Group, sendConcurrency)
}
