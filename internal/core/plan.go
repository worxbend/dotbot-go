package core

type Plan struct {
	Operations []Operation
}

type Operation struct {
	Directive string
	Target    string
	Detail    string
}
