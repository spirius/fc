package main

import (
	"flag"
	"fmt"
	"github.com/spirius/fc"
	"os"
)

type filter struct {
	name string
	args []string
}

type Config struct {
	filters       []*filter
	currentFilter *filter
}

type filterName Config
type filterArg Config

func (cc *filterName) Set(value string) error {
	c := (*Config)(cc)

	f := &filter{
		name: value,
	}

	c.filters = append(c.filters, f)
	c.currentFilter = f

	return nil
}

func (cc *filterName) String() string {
	return ""
}

func (cc *filterArg) Set(value string) error {
	c := (*Config)(cc)

	if c.currentFilter == nil {
		return fmt.Errorf("filter arguments specified before any filter definition")
	}

	c.currentFilter.args = append(c.currentFilter.args, value)

	return nil
}

func (cc *filterArg) String() string {
	return ""
}

func main() {
	c := &Config{}

	flagSet := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	flagSet.Var((*filterName)(c), "f", "filter name")
	flagSet.Var((*filterArg)(c), "a", "filter argument")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return
	}

	if len(c.filters) < 2 {
		panic("At least two filters are required")
	}

	pipeline := fc.DefaultFC.NewPipeline()

	var fn func(string, ...string) error

	for k, f := range c.filters {
		if k == 0 {
			fn = pipeline.SetInputFilter
		} else if k == (len(c.filters) - 1) {
			fn = pipeline.SetOutputFilter
		} else {
			fn = pipeline.AddFilter
		}

		if err := fn(f.name, f.args...); err != nil {
			panic(err)
		}
	}

	if err := pipeline.Process(os.Stdin, os.Stdout); err != nil {
		panic(err)
	}
}
