package web

// Middleware represents the signature for any middleware function
type Middleware func(Handler) Handler

func applyMiddlewares(handler Handler, mids ...Middleware) Handler {
	//need to apply the last one first
	for i := len(mids) - 1; i >= 0; i-- {
		mid := mids[i]
		if mid != nil {
			handler = mid(handler)
		}
	}
	return handler
}
