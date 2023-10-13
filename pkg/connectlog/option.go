package connectlog

type Option func(o *options)

type options struct {
	logSuccess bool
}

func WithSuccess() Option {
	return func(o *options) {
		o.logSuccess = true
	}
}
