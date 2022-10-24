// package korrel8 generic interfaces and algorithms to correlate objects between different domains.
//
// Each domain needs an implementation of the interfaces here.
package korrel8

import (
	"context"
	"net/url"
	"path"
	"time"
)

// Object represents an instance of a signal.
//
// Object has no methods to avoid clashes with fields or method names of the underlying object.
// The Class type provides some methods for inspecting objects.
// Object implementations MUST be pointers and MUST support JSON marshal/unmarshal.
type Object any

// Domain is a collection of classes describing signals in the same family.
//
// Domain implementations must be comparable.
type Domain interface {
	String() string        // Name of the domain
	Class(string) Class    // Find a class by name, return nil if not found.
	KnownClasses() []Class // List of known classes in the Domain
	NewQuery() Query       // NewQuery returns a new query, can be unmarshaled from JSON.
}

// Class identifies a subset of objects from the same domain with the same schema.
//
// For example Pod is a class in the k8s domain.
// Class implementations must be comparable.
type Class interface {
	Domain() Domain       // Domain of this class.
	New() Object          // Return a new instance of the class, can be unmarshaled from JSON.
	Contains(Object) bool // True if object is in this class
	Key(Object) any       // Comparable key for de-duplication or nil if object is not in this class.
	String() string       // Name of the class within the domain, e.g "Pod.v1". See ClassName()
}

// ClassName is the qualified domain/name of a class, e.g. "k8s/Pod.v1"
func ClassName(c Class) string { return path.Join(c.Domain().String(), c.String()) }

// Result gathers results from Store.Get calls.
type Result interface {
	Append(...Object)
}

// Query for result signals.
// Query implementations MUST be pointers and MUST support JSON marshal/unmarshal.
type Query interface {
	String() string                 // Plain query string
	Browser(base *url.URL) *url.URL // Browser URL for console
	REST(base *url.URL) *url.URL    // REST URL relative to base
}

// Store is a source of signals belonging to a single domain.
type Store interface {
	// Get executes one or more a Queries and appends objects to Result the resulting objects.
	Get(ctx context.Context, query Query, r Result) error
}

// Rule encapsulates logic to find correlated goal objects from a start object.
//
type Rule interface {
	Start() Class   // Class of start object
	Goal() Class    // Class of desired result object(s)
	String() string // Name of the rule

	// Apply the rule to start Object.
	// Return a list of queries for correlated objects in the Goal() domain.
	// The queries include the contraint (which can be nil)
	Apply(Object, *Constraint) (Query, error)
}

// Constraint to apply to the result of following a rule.
type Constraint struct {
	Start *time.Time // Include only results timestamped after this time.
	End   *time.Time // Include only results timestamped before this time.
}
