package cbhttp

func ContentLength(length int64) RequestOption {
	return func(r *Request) *Request {
		r.ContentLength = length
		return r
	}
}
