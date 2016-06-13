package pipeline

import (
	"io"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"fmt"
)

// Definition of a pipe
type Definition interface {
	CreatePipe(reqID string) Pipe
}

// DefinePipeFromReader returns a Pipe Definition as defined in the Reader
func DefinePipe(r io.Reader) (Definition, error) {
	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	type FittingDef map[string]interface{}
	type PipeDef map[string][]FittingDef

	var pipeConfig PipeDef
	err = yaml.Unmarshal(bytes, &pipeConfig)
	if err != nil {
		return nil, err
	}

	// todo: validation of structures

	var reqFittings []Fitting
	reqDefs := pipeConfig["request"]
	if reqDefs != nil {
		for _, fittingDef := range reqDefs {
			if len(fittingDef) > 1 {
				return nil, fmt.Errorf("bad structure")
			}
			for name, config := range fittingDef{
				fitting := NewFitting(name, config)
				handler := fitting.RequestHandlerFunc()
				if handler != nil {
					reqFittings = append(reqFittings, fitting)
				}
			}
		}
	}

	var resFittings []Fitting
	resDefs := pipeConfig["response"]
	if resDefs != nil {
		for _, fittingDef := range resDefs {
			if len(fittingDef) > 1 {
				return nil, fmt.Errorf("bad structure")
			}
			for name, config := range fittingDef{
				fitting := NewFitting(name, config)
				handler := fitting.ResponseHandlerFunc()
				if handler != nil {
					resFittings = append(resFittings, fitting)
				}
			}
		}
	}

	return NewDefinition(reqFittings, resFittings)
}

// DefinePipe returns a Pipe Definition defined by the passed handlers
func NewDefinition(reqFittings []Fitting, resFittings []Fitting) (Definition, error) {
	return &definition{reqFittings, resFittings}, nil
}

type definition struct {
	reqFittings []Fitting
	resFittings []Fitting
}

// if reqId is nil, will create and use an internal id
func (s *definition) CreatePipe(reqID string) Pipe {
	return newPipe(reqID, s.reqFittings, s.resFittings)
}
