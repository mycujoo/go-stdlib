package gcpconnect

import (
	"github.com/mycujoo/go-stdlib/pkg/connectlog"
	"google.golang.org/protobuf/encoding/protojson"
)

type Option func(o *options)

type options struct {
	logOptions     []connectlog.Option
	marshalOptions protojson.MarshalOptions
}

// WithLogOptions sets the options for the logging interceptor.
// E.g. WithLogOptions(connectlog.WithSuccess())
func WithLogOptions(opts ...connectlog.Option) Option {
	return func(o *options) {
		o.logOptions = append(o.logOptions, opts...)
	}
}

// WithJSONMarshalOptions sets the marshaling options for the JSON codec.
// Example: `WithJSONMarshalOptions(protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true})`
func WithJSONMarshalOptions(opts protojson.MarshalOptions) Option {
	return func(o *options) {
		o.marshalOptions = opts
	}
}
