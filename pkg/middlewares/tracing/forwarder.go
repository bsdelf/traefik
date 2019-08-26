package tracing

import (
	"context"
	"net/http"

	"github.com/containous/traefik/v2/pkg/middlewares"
	"github.com/containous/traefik/v2/pkg/tracing"
	"github.com/opentracing/opentracing-go/ext"
)

const (
	forwarderTypeName = "TracingForwarder"
)

type forwarderMiddleware struct {
	router  string
	service string
	next    http.Handler
}

// NewForwarder creates a new forwarder middleware that traces the outgoing request.
func NewForwarder(ctx context.Context, router, service string, next http.Handler) http.Handler {
	middlewares.GetLogger(ctx, "tracing", forwarderTypeName).
		Debugf("Added outgoing tracing middleware %s", service)

	return &forwarderMiddleware{
		router:  router,
		service: service,
		next:    next,
	}
}

func (f *forwarderMiddleware) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	tr, err := tracing.FromContext(req.Context())
	if err != nil {
		f.next.ServeHTTP(rw, req)
		return
	}

	opParts := []string{f.service, f.router}
	span, req, finish := tr.StartSpanf(req, ext.SpanKindRPCClientEnum, "forward", opParts, "/")
	defer finish()

	span.SetTag("service.name", f.service)
	span.SetTag("router.name", f.router)
	ext.HTTPMethod.Set(span, req.Method)
	ext.HTTPUrl.Set(span, req.URL.String())
	span.SetTag("http.host", req.Host)

	tracing.InjectRequestHeaders(req)

	recorder := newStatusCodeRecoder(rw, 200)

	f.next.ServeHTTP(recorder, req)

	tracing.LogResponseCode(span, recorder.Status())
}
