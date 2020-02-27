package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spirius/fc"

	"github.com/blang/semver"
	"github.com/juju/errors"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

func usage(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}
	fmt.Fprintf(os.Stderr, `gofc - structured data decoder/encoder

Usage:
gofc -i DECODER [ARG1, [...]] -o ENCODER [ARG1, [...]]

Options:
 -i            - input decoder
 -o            - output encoder
 -check-update - check if new version is available
 -self-update  - update to latest version

Supported coders:
json, j        - JSON decoder/encoder
yaml, yml, y   - YANL decoder/encoder
hcl, h         - HCL decoder/encoder, only HCL Attributes are supported, blocks are ignored
toml, t        - TOML decoder/encoder
null, n        - null decoder
tpl            - template encoder, provides golang template based engine
  path         - template file path (e.g.: gofc -i n -o tpl config.tpl)

For more information and examples, please visit https://github.com/spirius/fc

`)
	os.Exit(1)
}

type coderConfig struct {
	name string
	args []string
}

type config struct {
	decoder *coderConfig
	encoder *coderConfig
}

func readCoderConfig(c **coderConfig, args []string) ([]string, error) {
	conf := &coderConfig{}
	*c = conf
	if len(args) == 0 {
		return nil, errors.New("not enough arguments")
	}
	conf.name = args[0]
	args = args[1:]
	for k, v := range args {
		if len(v) > 0 && v[0] == '-' {
			return args[k:], nil
		}
		conf.args = append(conf.args, v)
	}
	return nil, nil
}

// Version is set during build.
var Version = "local-build"

func checkUpdate() (*selfupdate.Release, error) {
	latest, found, err := selfupdate.DetectLatest("spirius/fc")
	if err != nil {
		return nil, errors.Annotatef(err, "cannot check version")
	}

	v, verr := semver.Make(Version)

	if !found || (verr == nil && latest.Version.LTE(v)) {
		fmt.Printf("Current version is the latest: %s\n", Version)
		return nil, nil
	}

	latest.ReleaseNotes = strings.TrimSpace(latest.ReleaseNotes)

	return latest, nil
}

func selfUpdate() error {
	latest, err := checkUpdate()
	if err != nil {
		return errors.Trace(err)
	} else if latest == nil {
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return errors.Annotatef(err, "cannot locate executable path")
	}
	if err := selfupdate.UpdateTo(latest.AssetURL, exe); err != nil {
		return errors.Annotatef(err, "cannot update binary")
	}

	fmt.Println("Successfully updated to version", latest.Version)
	fmt.Println("Release note:\n", latest.ReleaseNotes)

	return nil
}

func main() {
	var conf config
	var err error

	args := os.Args[1:]
	for len(args) > 0 {
		e := args[0]
		switch e {
		case "-h", "--help":
			usage(nil)
		case "-v", "--version":
			fmt.Printf("Version: %s\n", Version)
			os.Exit(0)
		case "-i":
			if conf.decoder != nil {
				usage(errors.New("input decoder is already set"))
			}
			args, err = readCoderConfig(&conf.decoder, args[1:])
		case "-o":
			if conf.encoder != nil {
				usage(errors.New("output encoder is already set"))
			}
			args, err = readCoderConfig(&conf.encoder, args[1:])
		case "-self-update":
			err = selfUpdate()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s\n", err)
			}
			return
		case "-check-update":
			latest, err := checkUpdate()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s\n", err)
			} else if latest != nil {
				fmt.Printf("New version is available: %s, current version: %s\n", latest.Version, Version)
				fmt.Printf("Release note:\n%s\n", latest.ReleaseNotes)
			}
			return
		default:
			usage(errors.Errorf("unknown argument '%s'", e))
		}
		if err != nil {
			usage(errors.Trace(err))
		}
	}

	if conf.decoder == nil {
		usage(errors.New("input decoder is not set"))
	}
	if conf.encoder == nil {
		usage(errors.New("output encoder is not set"))
	}

	cConf := &fc.Config{
		Decoder:     conf.decoder.name,
		DecoderArgs: conf.decoder.args,
		Encoder:     conf.encoder.name,
		EncoderArgs: conf.encoder.args,
		Input:       os.Stdin,
		Output:      os.Stdout,
	}

	err = fc.DefaultRecoder.Run(cConf)
	if err == nil {
		return
	}
	var trace []string
	traceableError, ok := err.(*errors.Err)

	if ok {
		err = errors.Cause(traceableError)
		trace = traceableError.StackTrace()
	}

	if argErr, ok := err.(*fc.ArgumentError); ok {
		usage(argErr)
	}

	fmt.Fprintln(os.Stderr, err)
	if trace != nil {
		fmt.Fprintf(os.Stderr, "%s\n", strings.Join(trace[1:], "\n"))
	}
	os.Exit(1)
}
