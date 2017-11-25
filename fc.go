package fc

import (
	"fmt"
	"io"
)

type FC struct {
	inputFilters  map[string]InputFilter
	filters       map[string]Filter
	outputFilters map[string]OutputFilter
}

type BaseFilter interface {
	Description() string
	setFC(*FC)
}

type baseFilter struct{}

func (baseFilter) setFC(*FC) {}

type InputFilter interface {
	BaseFilter
	Input(input io.Reader, output interface{}, args ...string) error
}

type Filter interface {
	BaseFilter
	Filter(input interface{}, args ...string) (interface{}, error)
}

type OutputFilter interface {
	BaseFilter
	Output(output io.Writer, input interface{}, args ...string) error
}

type Pipeline struct {
	fc *FC

	inputFilter InputFilter
	inputArgs   []string

	filters    []Filter
	filterArgs [][]string

	outputFilter OutputFilter
	outputArgs   []string
}

func NewFC() *FC {
	return &FC{
		inputFilters:  make(map[string]InputFilter),
		filters:       make(map[string]Filter),
		outputFilters: make(map[string]OutputFilter),
	}
}

func (f *FC) AddInputFilter(filter InputFilter, names ...string) error {
	for _, name := range names {
		if _, ok := f.inputFilters[name]; ok {
			return fmt.Errorf("Cannot add InputFilter, filter '%s' already exists", name)
		}
	}
	for _, name := range names {
		f.inputFilters[name] = filter
	}
	filter.setFC(f)
	return nil
}

func (f *FC) AddFilter(filter Filter, names ...string) error {
	for _, name := range names {
		if _, ok := f.filters[name]; ok {
			return fmt.Errorf("Cannot add Filter, filter '%s' already exists", name)
		}
	}
	for _, name := range names {
		f.filters[name] = filter
	}
	filter.setFC(f)
	return nil
}

func (f *FC) AddOutputFilter(filter OutputFilter, names ...string) error {
	for _, name := range names {
		if _, ok := f.outputFilters[name]; ok {
			return fmt.Errorf("Cannot add OutputFilter, filter '%s' already exists", name)
		}
	}
	for _, name := range names {
		f.outputFilters[name] = filter
	}
	filter.setFC(f)
	return nil
}

func (f *FC) NewPipeline() *Pipeline {
	return &Pipeline{
		fc: f,
	}
}

func (f *FC) GetInputFilter(name string) (filter InputFilter, err error) {
	var ok bool
	filter, ok = f.inputFilters[name]

	if !ok {
		err = fmt.Errorf("Unknown input filter '%s'", name)
	}

	return
}

func (p *Pipeline) SetInputFilter(inputFilter string, args ...string) error {
	var ok bool
	if p.inputFilter, ok = p.fc.inputFilters[inputFilter]; !ok {
		return fmt.Errorf("Unknown InputFilter '%s'", inputFilter)
	}
	p.inputArgs = args
	return nil
}

func (p *Pipeline) SetOutputFilter(outputFilter string, args ...string) error {
	var ok bool
	if p.outputFilter, ok = p.fc.outputFilters[outputFilter]; !ok {
		return fmt.Errorf("Unknown OutputFilter '%s'", outputFilter)
	}
	p.outputArgs = args
	return nil
}

func (p *Pipeline) AddFilter(filterName string, args ...string) error {
	var (
		f  Filter
		ok bool
	)
	if f, ok = p.fc.filters[filterName]; !ok {
		return fmt.Errorf("Unknown Filter '%s'", filterName)
	}
	p.filters = append(p.filters, f)
	p.filterArgs = append(p.filterArgs, args)
	return nil
}

func (p *Pipeline) Process(in io.Reader, out io.Writer) error {
	var (
		data interface{}
		err  error
	)

	if err = p.inputFilter.Input(in, &data, p.inputArgs...); err != nil {
		return fmt.Errorf("Error while processing input: %s", err)
	}

	for k, filter := range p.filters {
		if data, err = filter.Filter(data, p.filterArgs[k]...); err != nil {
			return fmt.Errorf("Error while processing filter: %s", err)
		}
	}

	return p.outputFilter.Output(out, data, p.outputArgs...)
}

var DefaultFC = NewFC()
