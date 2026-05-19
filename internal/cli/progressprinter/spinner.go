// Copyright 2026. Triad National Security, LLC. All rights reserved.

package progressprinter

type spinner struct {
	index int
	chars []string
	stop  bool
	done  string
}

func newSpinner() *spinner {
	chars := []string{"\\", "|", "/", "-"}
	done := "+"

	return &spinner{
		chars: chars,
		done:  done,
	}
}

func (s *spinner) String() string {
	if s.stop {
		return s.done
	}
	s.index = (s.index + 1) % len(s.chars)
	return s.chars[s.index]
}

func (s *spinner) Stop() {
	s.stop = true
}
