package requestctx

import "context"

type requestIDKey struct{}
type callerKey struct{}
type callerTypeKey struct{}

// WithRequestID 将 request id 写入上下文。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// GetRequestID 读取 request id。
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey{}).(string); ok {
		return requestID
	}

	return ""
}

// WithCaller 将调用方身份写入上下文。
func WithCaller(ctx context.Context, caller string, callerType string) context.Context {
	ctx = context.WithValue(ctx, callerKey{}, caller)
	ctx = context.WithValue(ctx, callerTypeKey{}, callerType)

	return ctx
}

// GetCaller 读取调用方身份，未设置时返回 anonymous。
func GetCaller(ctx context.Context) (string, string) {
	caller, ok := ctx.Value(callerKey{}).(string)
	if !ok || caller == "" {
		caller = "anonymous"
	}

	callerType, ok := ctx.Value(callerTypeKey{}).(string)
	if !ok || callerType == "" {
		callerType = "anonymous"
	}

	return caller, callerType
}
