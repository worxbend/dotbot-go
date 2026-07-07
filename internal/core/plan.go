package core

// Plan is the previewable operation list produced from configuration tasks.
type Plan struct {
	// Operations contains planned directive work in execution order.
	Operations []Operation
}

// Operation describes one planned directive action.
type Operation struct {
	// Directive is the directive name, such as link, create, shell, or clean.
	Directive string
	// Target is the primary path or command affected by the operation.
	Target string
	// Detail carries optional directive-specific context for display or JSON output.
	Detail string
}
