package main

type Colorscheme struct {
	Identifier string

	Candidate struct {
		Normal   string
		Selected string
	}

	Fog struct {
		Text       string
		Background string
	}
}
