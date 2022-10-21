package fasthttp

import (
	"github.com/valyala/fasthttp"
)

type FilterFunc func(fasthttp.RequestHandler) fasthttp.RequestHandler

func FilterChain(filters ...FilterFunc) FilterFunc {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		for i := len(filters) - 1; i >= 0; i-- {
			next = filters[i](next)
		}
		return next
	}
}
