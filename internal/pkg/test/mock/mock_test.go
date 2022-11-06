package mock

import (
	"testing"

	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/stretchr/testify/assert"
)

func TestDomain(t *testing.T) {
	d := Domain("foo")
	assert.Equal(t, "foo", d.String())
	assert.Equal(t, Class("foo/x"), d.Class("x"))
	assert.Empty(t, d.Classes())

	d = Domain("foo a b c")
	assert.Equal(t, "foo", d.String())
	assert.Equal(t, Class("foo/a"), d.Class("a"))
	assert.Equal(t, nil, d.Class("x"))
	assert.Equal(t, []korrel8.Class{Class("foo/a"), Class("foo/b"), Class("foo/c")}, d.Classes())
}

func TestClass(t *testing.T) {
	c := Class("d/c")
	assert.Equal(t, Domain("d"), c.Domain())
	assert.Equal(t, "c", c.String())
	assert.Equal(t, Object("d/c:"), c.New())
	assert.Equal(t, Object("d/c:foo"), c.Key(Object("d/c:foo")))

	c = Class("c")
	assert.Equal(t, Domain(""), c.Domain())
	assert.Equal(t, Object("c:"), c.New())
	assert.Equal(t, Object("c:foo"), c.Key(Object("c:foo")))
}

func TestObject(t *testing.T) {
	o := Object("d/c:hello")
	assert.Equal(t, []any{Class("d/c"), "hello"}, []any{o.Class(), o.Data()})
}

func TestStore_Get(t *testing.T) {
	r := korrel8.NewListResult()

	Store{}.Get(nil, NewQuery("X/foo:x", "Y/bar.y", "foo:a", "bar:b", ":u", ":v"), r)
	want := NewObjects("X/foo:x", "Y/bar.y", "foo:a", "bar:b", ":u", ":v")
	assert.ElementsMatch(t, want, r.List())
}