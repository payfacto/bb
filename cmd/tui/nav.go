package tui

import "strings"

type navStack struct {
	views []View
}

func (s *navStack) Push(v View) {
	s.views = append(s.views, v)
}

func (s *navStack) Pop() View {
	if len(s.views) == 0 {
		return nil
	}
	top := s.views[len(s.views)-1]
	s.views = s.views[:len(s.views)-1]
	return top
}

func (s *navStack) Top() View {
	if len(s.views) == 0 {
		return nil
	}
	return s.views[len(s.views)-1]
}

func (s *navStack) Len() int {
	return len(s.views)
}

func (s *navStack) Breadcrumb() string {
	if len(s.views) == 0 {
		return ""
	}
	parts := make([]string, len(s.views))
	for i, v := range s.views {
		if i == len(s.views)-1 {
			parts[i] = breadcrumbActive.Render(v.Title())
		} else {
			parts[i] = breadcrumbStyle.Render(v.Title())
		}
	}
	return strings.Join(parts, breadcrumbSep.String())
}
