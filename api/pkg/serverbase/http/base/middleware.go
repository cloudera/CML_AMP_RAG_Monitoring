package sbhttpbase

type MiddlewareFunc func(request *Request, next HandleFunc)

func (fn MiddlewareFunc) Register(path, method string) MiddlewareFunc {
	return fn
}

var _ RegistrableMiddleware = MiddlewareFunc(nil)

type RegistrableMiddleware interface {
	Register(path, method string) MiddlewareFunc
}
