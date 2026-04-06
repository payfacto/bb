package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

const maxMRU = 5

// History persists per-workspace favourites, global MRU lists, and
// workspace-scoped list caches so the TUI never needs to re-fetch unless
// the user explicitly requests a refresh.
type History struct {
	Favourites        map[string][]string        `json:"favourites"`                   // workspace → []slug
	MRU               []MRUEntry                 `json:"mru"`                          // newest first, capped at maxMRU
	RepoCache         map[string][]CachedRepo    `json:"repo_cache,omitempty"`         // workspace → repos
	ProjectFavourites map[string][]string        `json:"project_favourites,omitempty"` // workspace → []key
	ProjectMRU        []ProjectMRUEntry          `json:"project_mru,omitempty"`        // newest first, capped at maxMRU
	ProjectCache      map[string][]CachedProject `json:"project_cache,omitempty"`      // workspace → projects
}

// MRUEntry is a recently-visited repository.
type MRUEntry struct {
	Workspace string `json:"workspace"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
}

// ProjectMRUEntry is a recently-visited project.
type ProjectMRUEntry struct {
	Workspace string `json:"workspace"`
	Key       string `json:"key"`
	Name      string `json:"name"`
}

// CachedRepo stores the minimal repo fields needed by the TUI.
type CachedRepo struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	IsPrivate bool   `json:"is_private"`
}

// CachedProject stores the minimal project fields needed by the TUI.
type CachedProject struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	IsPrivate bool   `json:"is_private"`
}

// HistoryPath returns the history file path derived from the config file path.
func HistoryPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), ".bbcloud_history.json")
}

// Load reads the history file. Returns an empty History (not an error) when the
// file does not exist yet.
func Load(path string) (*History, error) {
	h := &History{Favourites: make(map[string][]string)}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return h, nil
		}
		return h, err
	}
	if err := json.Unmarshal(data, h); err != nil {
		return h, fmt.Errorf("parse %s: %w", path, err)
	}
	if h.Favourites == nil {
		h.Favourites = make(map[string][]string)
	}
	return h, nil
}

// Save writes the history to path, creating the file if it does not exist.
func (h *History) Save(path string) error {
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// IsFavourite reports whether slug is starred in the given workspace.
func (h *History) IsFavourite(ws, slug string) bool {
	return slices.Contains(h.Favourites[ws], slug)
}

// ToggleFavourite adds slug to favourites if absent, or removes it if present.
func (h *History) ToggleFavourite(ws, slug string) {
	favs := h.Favourites[ws]
	for i, s := range favs {
		if s == slug {
			h.Favourites[ws] = append(favs[:i], favs[i+1:]...)
			return
		}
	}
	favs = append(favs, slug)
	slices.Sort(favs)
	h.Favourites[ws] = favs
}

// AddMRU prepends an entry for (ws, slug, name), deduplicates, and caps at maxMRU.
func (h *History) AddMRU(ws, slug, name string) {
	updated := make([]MRUEntry, 0, maxMRU)
	updated = append(updated, MRUEntry{Workspace: ws, Slug: slug, Name: name})
	for _, e := range h.MRU {
		if len(updated) >= maxMRU {
			break
		}
		if !(e.Workspace == ws && e.Slug == slug) {
			updated = append(updated, e)
		}
	}
	h.MRU = updated
}

// SetRepos stores the repo list for the given workspace in the cache.
func (h *History) SetRepos(ws string, repos []CachedRepo) {
	if h.RepoCache == nil {
		h.RepoCache = make(map[string][]CachedRepo)
	}
	h.RepoCache[ws] = repos
}

// Repos returns the cached repo list for the given workspace.
// Returns false when no cache entry exists for that workspace.
func (h *History) Repos(ws string) ([]CachedRepo, bool) {
	repos, ok := h.RepoCache[ws]
	return repos, ok && len(repos) > 0
}

// ClearRepos removes the cached repo list for the given workspace.
func (h *History) ClearRepos(ws string) {
	delete(h.RepoCache, ws)
}

// RecentSlugs returns the MRU slugs for the given workspace, newest first.
func (h *History) RecentSlugs(ws string) []string {
	var slugs []string
	for _, e := range h.MRU {
		if e.Workspace == ws {
			slugs = append(slugs, e.Slug)
		}
	}
	return slugs
}

// --- Project favourites, MRU, and cache ---

// IsProjectFavourite reports whether key is starred in the given workspace.
func (h *History) IsProjectFavourite(ws, key string) bool {
	if h.ProjectFavourites == nil {
		return false
	}
	return slices.Contains(h.ProjectFavourites[ws], key)
}

// ToggleProjectFavourite adds key to project favourites if absent, or removes it if present.
func (h *History) ToggleProjectFavourite(ws, key string) {
	if h.ProjectFavourites == nil {
		h.ProjectFavourites = make(map[string][]string)
	}
	favs := h.ProjectFavourites[ws]
	for i, k := range favs {
		if k == key {
			h.ProjectFavourites[ws] = append(favs[:i], favs[i+1:]...)
			return
		}
	}
	favs = append(favs, key)
	slices.Sort(favs)
	h.ProjectFavourites[ws] = favs
}

// AddProjectMRU prepends an entry for (ws, key, name), deduplicates, and caps at maxMRU.
func (h *History) AddProjectMRU(ws, key, name string) {
	updated := make([]ProjectMRUEntry, 0, maxMRU)
	updated = append(updated, ProjectMRUEntry{Workspace: ws, Key: key, Name: name})
	for _, e := range h.ProjectMRU {
		if len(updated) >= maxMRU {
			break
		}
		if !(e.Workspace == ws && e.Key == key) {
			updated = append(updated, e)
		}
	}
	h.ProjectMRU = updated
}

// RecentProjectKeys returns the MRU project keys for the given workspace, newest first.
func (h *History) RecentProjectKeys(ws string) []string {
	var keys []string
	for _, e := range h.ProjectMRU {
		if e.Workspace == ws {
			keys = append(keys, e.Key)
		}
	}
	return keys
}

// SetProjects stores the project list for the given workspace in the cache.
func (h *History) SetProjects(ws string, projects []CachedProject) {
	if h.ProjectCache == nil {
		h.ProjectCache = make(map[string][]CachedProject)
	}
	h.ProjectCache[ws] = projects
}

// Projects returns the cached project list for the given workspace.
// Returns false when no cache entry exists or the slice is empty.
func (h *History) Projects(ws string) ([]CachedProject, bool) {
	projects, ok := h.ProjectCache[ws]
	return projects, ok && len(projects) > 0
}

// ClearProjects removes the cached project list for the given workspace.
func (h *History) ClearProjects(ws string) {
	delete(h.ProjectCache, ws)
}
