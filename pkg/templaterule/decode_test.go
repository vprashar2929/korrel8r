package templaterule

import (
	"strings"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecode(t *testing.T) {
	foo := mock.Domain("foo a z")
	domains := map[string]korrel8r.Domain{"foo": foo}
	a, z := foo.Class("a"), foo.Class("z")

	r := strings.NewReader(`
groups:
  - name: wild
    classes: [bill, hickok]
rules:
  - name:   one
    start:  {domain: "foo", classes: [a]}
    goal:   {domain: "foo", classes: [z]}
    result: {query: dummy, class: dummy}
  - name:   two
    start:  {domain: "foo", classes: [a]}
    goal:   {domain: "foo", classes: [z]}
    result: {query: dummy, class: dummy}
`)

	rules, err := Decode(r, domains, nil)
	require.NoError(t, err)
	want := []mock.Rule{mockRule("one", a, z), mockRule("two", a, z)}
	assert.Equal(t, want, mockRules(rules...))
}

func TestExpand(t *testing.T) {
	g := NewGroups([]Group{
		{Name: "foo", Classes: []string{"f1", "f2"}},
		{Name: "bar", Classes: []string{"b0"}},
		{Name: "both", Classes: []string{"foo", "bar"}},
		{Name: "more", Classes: []string{"both", "m1"}},
	})
	got := g.Expand([]string{"a", "b", "foo", "c", "bar"})
	want := []string{"a", "b", "f1", "f2", "c", "b0"}
	assert.Equal(t, want, got)

	got = g.Expand([]string{"both"})
	want = []string{"f1", "f2", "b0"}
	assert.Equal(t, want, got)

	got = g.Expand([]string{"more"})
	want = []string{"f1", "f2", "b0", "m1"}
	assert.Equal(t, want, got)
}
