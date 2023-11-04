// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package engine implements generic correlation logic to correlate across domains.
package engine

import (
	"context"
	"fmt"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

var log = logging.Log()

// Engine combines a set of domains and a set of rules, so it can perform correlation.
type Engine struct {
	domains       []korrel8r.Domain
	domainMap     map[string]korrel8r.Domain
	stores        map[string][]korrel8r.Store
	storeConfigs  map[string][]korrel8r.StoreConfig
	rules         []korrel8r.Rule
	templateFuncs map[string]any
}

func New(domains ...korrel8r.Domain) *Engine {
	e := &Engine{
		domains:       slices.Clone(domains), // Predicatable order for Domains()
		domainMap:     map[string]korrel8r.Domain{},
		stores:        map[string][]korrel8r.Store{},
		storeConfigs:  map[string][]korrel8r.StoreConfig{},
		templateFuncs: map[string]any{},
	}
	for _, d := range domains {
		e.domainMap[d.Name()] = d
		e.addTemplateFuncs(d)
	}
	return e
}

// Domain returns the named domain or nil if not found.
func (e *Engine) Domain(name string) korrel8r.Domain { return e.domainMap[name] }
func (e *Engine) Domains() []korrel8r.Domain         { return e.domains }
func (e *Engine) DomainErr(name string) (korrel8r.Domain, error) {
	if d := e.Domain(name); d != nil {
		return d, nil
	}
	return nil, korrel8r.DomainNotFoundErr{Domain: name}
}

// StoresFor returns the known stores for a domain.
func (e *Engine) StoresFor(d korrel8r.Domain) []korrel8r.Store { return e.stores[d.Name()] }

// StoreConfigsFor returns store configurations added with AddStoreConfig
func (e *Engine) StoreConfigsFor(d korrel8r.Domain) []korrel8r.StoreConfig {
	return e.storeConfigs[d.Name()]
}

// StoreErr returns the default (first) store for domain, or an error.
func (e *Engine) StoreErr(d korrel8r.Domain) (korrel8r.Store, error) {
	stores := e.StoresFor(d)
	if len(stores) == 0 {
		return nil, korrel8r.StoreNotFoundErr{Domain: d}
	}
	return stores[0], nil
}

// TemplateFuncser can be implemented by Domain or Store implementations to contribute
// domain-specific template functions to template rules generated by the Engine.
// See text/template.Template.Funcs for details.
type TemplateFuncser interface{ TemplateFuncs() map[string]any }

// AddStore adds a store to the engine.
func (e *Engine) AddStore(s korrel8r.Store) error {
	domain := s.Domain().Name()
	e.stores[domain] = append(e.stores[domain], s)
	e.addTemplateFuncs(s)
	return nil
}

// AddStoreConfig creates a store from configuration and adds it to the engine.
//
// If there is an error, it is returned, and the configuration is stored with the error field set.
func (e *Engine) AddStoreConfig(sc korrel8r.StoreConfig) (err error) {
	d, err := e.DomainErr(sc[korrel8r.StoreKeyDomain])
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			sc[korrel8r.StoreKeyError] = err.Error()
		}
		e.storeConfigs[d.Name()] = append(e.storeConfigs[d.Name()], sc)
	}()
	store, err := d.Store(sc)
	if err != nil {
		return err
	}
	if err := e.AddStore(store); err != nil {
		return err
	}
	return nil
}

func (e *Engine) addTemplateFuncs(v any) {
	// Stores and Domains may implement TemplateFuncser if they provide template helper functions for rules
	if tf, ok := v.(TemplateFuncser); ok {
		maps.Copy(e.templateFuncs, tf.TemplateFuncs())
	}
}

// Class parses a full class name and returns the
func (e *Engine) Class(fullname string) (korrel8r.Class, error) {
	d, c, ok := korrel8r.SplitClassName(fullname)
	if !ok {
		return nil, fmt.Errorf("invalid class name: %v", fullname)
	} else {
		return e.DomainClass(d, c)
	}
}

func (e *Engine) DomainClass(domain, class string) (korrel8r.Class, error) {
	d, err := e.DomainErr(domain)
	if err != nil {
		return nil, err
	}
	c := d.Class(class)
	if c == nil {
		return nil, korrel8r.ClassNotFoundErr{Class: class, Domain: d}
	}
	return c, nil
}

// Query parses a query string to a query object.
func (e *Engine) Query(query string) (korrel8r.Query, error) {
	d, _, _, ok := korrel8r.SplitClassData(query)
	if !ok {
		return nil, fmt.Errorf("invalid query string: %v", query)
	}
	domain, err := e.DomainErr(d)
	if err != nil {
		return nil, err
	}
	return domain.Query(query)
}

func (e *Engine) Rules() []korrel8r.Rule { return e.rules }

func (e *Engine) AddRules(rules ...korrel8r.Rule) { e.rules = append(e.rules, rules...) }

// Graph creates a new graph of the rules and classes of this engine.
func (e *Engine) Graph() *graph.Graph { return graph.NewData(e.rules...).NewGraph() }

// TemplateFuncs returns template helper functions for stores and domains known to this engine.
// See text/template.Template.Funcs
func (e *Engine) TemplateFuncs() map[string]any { return e.templateFuncs }

// Get finds the store for the query.Class() and gets into result.
func (e *Engine) Get(ctx context.Context, class korrel8r.Class, query korrel8r.Query, result korrel8r.Appender) error {
	for _, store := range e.StoresFor(class.Domain()) {
		if err := store.Get(ctx, query, result); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) Follower(ctx context.Context) *Follower { return &Follower{Engine: e, Context: ctx} }
