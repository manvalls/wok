package wok

import (
	"net/http"
	"sync"

	"github.com/manvalls/wit"
)

type baseCR struct {
	triggerCb  func(string)
	reloadOnCb func(string)

	mux *sync.Mutex

	controller string
	socket     bool
	params     Params

	reloadOn []string

	addHeaders http.Header
	setHeaders http.Header
	status     int

	redirectLocation string
	redirectStatus   int

	actions        []interface{}
	cleanup        [][]interface{}
	currentCleanup []interface{}
}

type deltaAction struct {
	delta wit.Delta
}

type triggerAction struct {
	events []string
}

func newBaseCR() *baseCR {
	return &baseCR{
		mux:        &sync.Mutex{},
		reloadOn:   []string{},
		addHeaders: http.Header{},
		setHeaders: http.Header{},

		actions:        []interface{}{},
		cleanup:        [][]interface{}{},
		currentCleanup: []interface{}{},
	}
}

func (cr *baseCR) Controller() string {
	return cr.controller
}

func (cr *baseCR) Socket() bool {
	return cr.socket
}

func (cr *baseCR) Params() Params {
	return cr.params
}

func (cr *baseCR) Redirect(url string, status int) ControllerRequest {
	cr.mux.Lock()
	defer cr.mux.Unlock()

	cr.redirectLocation = url
	cr.redirectStatus = status
	return cr
}

func (cr *baseCR) SetStatus(status int) ControllerRequest {
	cr.mux.Lock()
	defer cr.mux.Unlock()

	cr.status = status
	return cr
}

func (cr *baseCR) AddHeader(key string, values ...string) ControllerRequest {
	cr.mux.Lock()
	defer cr.mux.Unlock()

	cr.addHeaders[key] = append(cr.addHeaders[key], values...)
	return cr
}

func (cr *baseCR) SetHeader(key string, values ...string) ControllerRequest {
	cr.mux.Lock()
	defer cr.mux.Unlock()

	delete(cr.addHeaders, key)
	cr.setHeaders[key] = values
	return cr
}

func (cr *baseCR) SendDelta(delta wit.Delta) ControllerRequest {
	cr.mux.Lock()
	defer cr.mux.Unlock()

	if len(cr.currentCleanup) > 0 {
		cr.cleanup = append(cr.cleanup, cr.currentCleanup)
		cr.currentCleanup = []interface{}{}
	}

	cr.actions = append(cr.actions, deltaAction{delta})
	return cr
}

func (cr *baseCR) ReloadOn(events ...string) ControllerRequest {
	cr.mux.Lock()
	defer cr.mux.Unlock()

	cr.reloadOn = append(cr.reloadOn, events...)

	if cr.reloadOnCb != nil {
		for _, event := range events {
			cr.reloadOnCb(event)
		}
	}

	return cr
}

func (cr *baseCR) Trigger(events ...string) ControllerRequest {
	cr.mux.Lock()
	defer cr.mux.Unlock()

	cr.actions = append(cr.actions, triggerAction{events})

	if cr.triggerCb != nil {
		for _, event := range events {
			cr.triggerCb(event)
		}
	}

	return cr
}

type baseCleanup struct {
	cr *baseCR
}

func (c baseCleanup) SendDelta(delta wit.Delta) Cleanup {
	c.cr.mux.Lock()
	defer c.cr.mux.Unlock()

	c.cr.currentCleanup = append(c.cr.currentCleanup, deltaAction{delta})
	return c
}

func (c baseCleanup) Trigger(events ...string) Cleanup {
	c.cr.mux.Lock()
	defer c.cr.mux.Unlock()

	c.cr.currentCleanup = append(c.cr.currentCleanup, triggerAction{events})
	return c
}

func (cr *baseCR) Cleanup() Cleanup {
	return baseCleanup{cr}
}

func (cr *baseCR) cleanupActions() []interface{} {
	cr.mux.Lock()
	defer cr.mux.Unlock()

	actions := append([]interface{}{}, cr.currentCleanup...)
	for i := len(cr.cleanup) - 1; i >= 0; i-- {
		actions = append(actions, cr.cleanup[i]...)
	}

	return actions
}
