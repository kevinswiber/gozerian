package pipeline

import (
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
)

// Pipe runs a series of request and response handlers - one created per request
type Pipe interface {
	ControlHolder
	RequestHandlerFunc() http.HandlerFunc
	ResponseHandlerFunc() ResponseHandlerFunc
}

var reqCounter int64

func newPipe(reqID string, reqFittings []Fitting, resFittings []Fitting) Pipe {

	// provide a default (transient) implementation of an ID
	if reqID == "" {
		reqID = string(strconv.FormatInt(atomic.AddInt64(&reqCounter, 1), 10))
	}

	return &pipe{
		reqID:    reqID,
		reqFittings: reqFittings,
		resFittings: resFittings,
	}
}

type pipe struct {
	reqID    string
	reqFittings []Fitting
	resFittings []Fitting
	control  Control
	writer   responseWriter
}

func (p *pipe) Control() Control {
	return p.control
}

func (p *pipe) RequestHandlerFunc() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		writer := p.setWriter(w, r)
		defer recoveryFunc(p.control)

		for _, fitting := range p.reqFittings {
			if p.control.Cancelled() {
				break
			}
			p.control.Log().Debugf("Enter req handler %s", fitting.ID())
			fitting.RequestHandlerFunc()(writer, r)
			p.control.Log().Debugf("Exit  req handler %s", fitting.ID())
		}
	}
}

func (p *pipe) ResponseHandlerFunc() ResponseHandlerFunc {

	return func(w http.ResponseWriter, r *http.Request, res *http.Response) {

		writer := p.setWriter(w, r)
		defer recoveryFunc(p.control)

		for _, fitting := range p.resFittings {
			if p.control.Cancelled() {
				break
			}
			p.control.Log().Debugf("Enter res handler %s", fitting.ID())
			fitting.ResponseHandlerFunc()(writer, r, res)
			p.control.Log().Debugf("Enter res handler %s", fitting.ID())
		}
	}
}

func (p *pipe) setWriter(w http.ResponseWriter, r *http.Request) responseWriter {

	writer, ok := w.(responseWriter)
	if !ok {
		f := logrus.Fields{
			"reqID":  p.reqID,
			"uri": r.RequestURI,
		}
		log := GetLogger().WithFields(f)

		ctx, cancel := context.WithTimeout(context.Background(), GetConfig().GetDuration(ConfigTimeout))
		p.control = NewControl(p.reqID, ctx, w, config, log, cancel)

		writer = newResponseWriter(w, p.control)
	}
	p.writer = writer
	return writer
}

func recoveryFunc(pc Control) {
	if r := recover(); r != nil {
		err := fmt.Errorf("%s", r)
		pc.Log().Warn("Panic Recovery Error: ", err)
		pc.SendError(err)
	}
}
