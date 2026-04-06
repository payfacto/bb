package tui

import "testing"

func TestListCache_Miss(t *testing.T) {
	c := newListCache()
	if _, ok := c.Get("missing"); ok {
		t.Error("expected miss on empty cache")
	}
}

func TestListCache_PinAndGet(t *testing.T) {
	c := newListCache()
	items := []listItem{{id: "1", title: "Repo A"}}
	c.Pin("k", items)

	got, ok := c.Get("k")
	if !ok {
		t.Fatal("expected hit after Pin")
	}
	if len(got) != 1 || got[0].id != "1" {
		t.Errorf("unexpected items: %v", got)
	}
}

func TestListCache_Invalidate(t *testing.T) {
	c := newListCache()
	c.Pin("k", []listItem{{title: "x"}})
	c.Invalidate("k")

	if _, ok := c.Get("k"); ok {
		t.Error("expected miss after Invalidate")
	}
}

func TestListCache_InvalidateMissingKey(t *testing.T) {
	c := newListCache()
	c.Invalidate("never-set") // must not panic
}

func TestListCache_IndependentKeys(t *testing.T) {
	c := newListCache()
	c.Pin("a", []listItem{{title: "A"}})
	c.Pin("b", []listItem{{title: "B"}})
	c.Invalidate("a")

	if _, ok := c.Get("a"); ok {
		t.Error("expected a to be invalidated")
	}
	if _, ok := c.Get("b"); !ok {
		t.Error("expected b to still be present")
	}
}
