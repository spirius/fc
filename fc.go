package fc

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/juju/errors"
)

// Config is the Recoder configuration
type Config struct {
	Decoder     string
	DecoderArgs []string

	Encoder     string
	EncoderArgs []string

	Input  io.Reader
	Output io.Writer
}

// Recoder represent set of encoders
// and decoders which can be combined
// to re-code structured data into
// one of supported formats.
type Recoder struct {
	Encoders map[string]Encoder
	Decoders map[string]Decoder

	Coders map[string]Coder
}

// Register new converter
func (r *Recoder) Register(c Coder) {
	for _, name := range c.Names() {
		if inp, ok := c.(Decoder); ok {
			r.Decoders[name] = inp
		}
		if out, ok := c.(Encoder); ok {
			r.Encoders[name] = out
		}
		r.Coders[name] = c
	}
}

// Initialize converters after registration
func (r *Recoder) Initialize() error {
	for _, c := range r.Coders {
		if err := c.Initialize(); err != nil {
			return errors.Annotate(err, "cannot initialize converter")
		}
	}
	return nil
}

// Run converter with provided configuration
func (r *Recoder) Run(config *Config) error {
	data, metadata, err := r.Decode(config)
	if err != nil {
		return errors.Annotatef(err, "cannot run input converter")
	}
	return errors.Annotatef(r.Encode(config, data, metadata), "cannot run output conveter")
}

// Decode function decodes data stream from config.Input
// using config.Decoder.
func (r *Recoder) Decode(config *Config) (interface{}, interface{}, error) {
	input, ok := r.Decoders[config.Decoder]
	if !ok {
		return nil, nil, errors.Errorf("unknown decoder '%s'", config.Decoder)
	}
	data, metadata, err := input.Decode(config.Input, config.DecoderArgs)
	if err != nil {
		return nil, nil, errors.Annotate(err, "error while processing input data")
	}
	return data, metadata, nil
}

// Encode function encodes data into config.Output stream
// using config.Encoder.
func (r *Recoder) Encode(config *Config, data interface{}, metadata interface{}) error {
	output, ok := r.Encoders[config.Encoder]
	if !ok {
		return errors.Errorf("unknown output type '%s'", config.Encoder)
	}
	if err := output.Encode(config.Output, data, metadata, config.EncoderArgs); err != nil {
		return errors.Annotate(err, "error while processing output data")
	}
	return nil
}

// Coder is the common interface for encoders and decoders.
type Coder interface {
	Initialize() error
	Names() []string
}

// Decoder interface.
type Decoder interface {
	Coder
	Decode(reader io.Reader, args []string) (interface{}, interface{}, error)
}

// Encoder interface.
type Encoder interface {
	Coder
	Encode(writer io.Writer, in interface{}, metadata interface{}, args []string) error
}

// ArgumentError is used, when convter function detects argument error
type ArgumentError struct {
	error string
}

func (e ArgumentError) Error() string {
	return e.error
}

// DefaultRecoder is a recoder with all built-in encoders and decoders.
var DefaultRecoder *Recoder

func init() {
	DefaultRecoder = &Recoder{
		Decoders: map[string]Decoder{},
		Encoders: map[string]Encoder{},
		Coders:   map[string]Coder{},
	}
	DefaultRecoder.Register(&coderJSON{})
	DefaultRecoder.Register(&coderYAML{})
	DefaultRecoder.Register(&coderHCL{})
	DefaultRecoder.Register(&coderTOML{})
	DefaultRecoder.Register(&coderNULL{})

	sess := session.New() //nolint
	tpl := newCoderTPL(DefaultRecoder, s3.New(sess))
	DefaultRecoder.Register(tpl)
	if err := DefaultRecoder.Initialize(); err != nil {
		panic(fmt.Sprintf("error: cannot initialize default recoder, %s", err))
	}
}
