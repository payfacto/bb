package tui

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/internal/history"
	"github.com/payfacto/bb/pkg/bitbucket"
)

// --- abbrevHash ---

func TestAbbrevHash_LongHash(t *testing.T) {
	h := "abcdef1234567890"
	got := abbrevHash(h)
	if got != h[:shortHashLen] {
		t.Errorf("expected %q, got %q", h[:shortHashLen], got)
	}
}

func TestAbbrevHash_ExactLength(t *testing.T) {
	h := "abcdef12" // exactly shortHashLen chars
	if abbrevHash(h) != h {
		t.Errorf("expected unchanged, got %q", abbrevHash(h))
	}
}

func TestAbbrevHash_Short(t *testing.T) {
	h := "abc"
	if abbrevHash(h) != h {
		t.Errorf("expected unchanged, got %q", abbrevHash(h))
	}
}

func TestAbbrevHash_Empty(t *testing.T) {
	if abbrevHash("") != "" {
		t.Error("expected empty string unchanged")
	}
}

// --- truncateStr ---

func TestTruncateStr_Short(t *testing.T) {
	s := "hello"
	if truncateStr(s, 10) != s {
		t.Errorf("expected unchanged, got %q", truncateStr(s, 10))
	}
}

func TestTruncateStr_ExactLimit(t *testing.T) {
	s := "hello"
	if truncateStr(s, 5) != s {
		t.Errorf("expected unchanged at exact limit, got %q", truncateStr(s, 5))
	}
}

func TestTruncateStr_Long(t *testing.T) {
	s := "hello world"
	got := truncateStr(s, 6)
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected ellipsis suffix, got %q", got)
	}
	if len([]rune(got)) != 6 {
		t.Errorf("expected 6 runes, got %d in %q", len([]rune(got)), got)
	}
}

// --- repoBaseTitle ---

func TestRepoBaseTitle_Private(t *testing.T) {
	r := bitbucket.Repo{Name: "MyRepo", IsPrivate: true}
	if repoBaseTitle(r) != "MyRepo" {
		t.Errorf("private repo should have no suffix, got %q", repoBaseTitle(r))
	}
}

func TestRepoBaseTitle_Public(t *testing.T) {
	r := bitbucket.Repo{Name: "MyRepo", IsPrivate: false}
	got := repoBaseTitle(r)
	if !strings.Contains(got, "public") {
		t.Errorf("public repo should have [public] suffix, got %q", got)
	}
}

// --- sortRepoItems ---

func makeSortTestItems() ([]listItem, *history.History) {
	repos := []bitbucket.Repo{
		{Slug: "zebra", Name: "Zebra"},
		{Slug: "apple", Name: "Apple"},
		{Slug: "mango", Name: "Mango"},
		{Slug: "berry", Name: "Berry"},
	}
	items := make([]listItem, len(repos))
	for i, r := range repos {
		items[i] = listItem{title: r.Name, data: r}
	}
	hist := &history.History{Favourites: make(map[string][]string)}
	return items, hist
}

func TestSortRepoItems_NoFavNoMRU(t *testing.T) {
	items, hist := makeSortTestItems()
	sorted := sortRepoItems(items, hist, "ws")

	// Expect A-Z: Apple, Berry, Mango, Zebra
	want := []string{"Apple", "Berry", "Mango", "Zebra"}
	for i, w := range want {
		if sorted[i].data.(bitbucket.Repo).Name != w {
			t.Errorf("pos %d: expected %q, got %q", i, w, sorted[i].data.(bitbucket.Repo).Name)
		}
	}
}

func TestSortRepoItems_FavouritesFirst(t *testing.T) {
	items, hist := makeSortTestItems()
	hist.ToggleFavourite("ws", "zebra")
	hist.ToggleFavourite("ws", "mango")

	sorted := sortRepoItems(items, hist, "ws")

	// Favs A-Z first: Mango, Zebra
	if sorted[0].data.(bitbucket.Repo).Slug != "mango" {
		t.Errorf("expected mango first fav, got %q", sorted[0].data.(bitbucket.Repo).Slug)
	}
	if sorted[1].data.(bitbucket.Repo).Slug != "zebra" {
		t.Errorf("expected zebra second fav, got %q", sorted[1].data.(bitbucket.Repo).Slug)
	}
	// Rest A-Z after
	if sorted[2].data.(bitbucket.Repo).Slug != "apple" {
		t.Errorf("expected apple third, got %q", sorted[2].data.(bitbucket.Repo).Slug)
	}
}

func TestSortRepoItems_MRUAfterFavs(t *testing.T) {
	items, hist := makeSortTestItems()
	hist.ToggleFavourite("ws", "apple")
	hist.AddMRU("ws", "zebra", "Zebra") // oldest MRU
	hist.AddMRU("ws", "berry", "Berry") // newest MRU

	sorted := sortRepoItems(items, hist, "ws")

	// apple is fav → first
	if sorted[0].data.(bitbucket.Repo).Slug != "apple" {
		t.Errorf("expected apple (fav) first, got %q", sorted[0].data.(bitbucket.Repo).Slug)
	}
	// berry is newest MRU → second
	if sorted[1].data.(bitbucket.Repo).Slug != "berry" {
		t.Errorf("expected berry (newest MRU) second, got %q", sorted[1].data.(bitbucket.Repo).Slug)
	}
	// zebra is older MRU → third
	if sorted[2].data.(bitbucket.Repo).Slug != "zebra" {
		t.Errorf("expected zebra (older MRU) third, got %q", sorted[2].data.(bitbucket.Repo).Slug)
	}
	// mango is rest → last
	if sorted[3].data.(bitbucket.Repo).Slug != "mango" {
		t.Errorf("expected mango (rest) last, got %q", sorted[3].data.(bitbucket.Repo).Slug)
	}
}

func TestSortRepoItems_FavMarker(t *testing.T) {
	items, hist := makeSortTestItems()
	hist.ToggleFavourite("ws", "apple")

	sorted := sortRepoItems(items, hist, "ws")

	if !strings.HasPrefix(sorted[0].title, repoFavMarker) {
		t.Errorf("fav item should have marker prefix, got %q", sorted[0].title)
	}
	if strings.HasPrefix(sorted[1].title, repoFavMarker) {
		t.Errorf("non-fav item should not have marker, got %q", sorted[1].title)
	}
}

func TestSortRepoItems_MRUMarker(t *testing.T) {
	items, hist := makeSortTestItems()
	hist.AddMRU("ws", "mango", "Mango")

	sorted := sortRepoItems(items, hist, "ws")

	var mangoItem listItem
	for _, it := range sorted {
		if it.data.(bitbucket.Repo).Slug == "mango" {
			mangoItem = it
			break
		}
	}
	if !strings.HasPrefix(mangoItem.title, repoMRUMarker) {
		t.Errorf("MRU item should have marker prefix, got %q", mangoItem.title)
	}
}

func TestSortRepoItems_Idempotent(t *testing.T) {
	items, hist := makeSortTestItems()
	hist.ToggleFavourite("ws", "berry")

	first := sortRepoItems(items, hist, "ws")
	second := sortRepoItems(first, hist, "ws") // sort already-sorted+marked items

	for i := range first {
		if first[i].data.(bitbucket.Repo).Slug != second[i].data.(bitbucket.Repo).Slug {
			t.Errorf("sort not idempotent at pos %d: %q vs %q",
				i, first[i].data.(bitbucket.Repo).Slug, second[i].data.(bitbucket.Repo).Slug)
		}
	}
}
