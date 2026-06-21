package app

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"dotbot-go/internal/core"
)

type planDocument struct {
	TaskCount       int             `json:"task_count"`
	ConfigFileCount int             `json:"config_file_count"`
	OperationCount  int             `json:"operation_count"`
	Base            string          `json:"base"`
	Operations      []planOperation `json:"operations"`
}

type planOperation struct {
	Directive string `json:"directive"`
	Target    string `json:"target"`
	Detail    string `json:"detail"`
}

func writePlanOutput(
	w io.Writer,
	format string,
	plan core.Plan,
	taskCount int,
	configCount int,
	base string,
) error {
	doc := newPlanDocument(plan, taskCount, configCount, base)
	switch format {
	case "", "text":
		_, err := io.WriteString(w, formatTextPlan(doc))
		return err
	case "json":
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		return encoder.Encode(doc)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func newPlanDocument(plan core.Plan, taskCount, configCount int, base string) planDocument {
	operations := make([]planOperation, 0, len(plan.Operations))
	for _, operation := range plan.Operations {
		operations = append(operations, planOperation{
			Directive: operation.Directive,
			Target:    operation.Target,
			Detail:    operation.Detail,
		})
	}
	return planDocument{
		TaskCount:       taskCount,
		ConfigFileCount: configCount,
		OperationCount:  len(operations),
		Base:            base,
		Operations:      operations,
	}
}

func formatTextPlan(doc planDocument) string {
	var b strings.Builder
	fmt.Fprintf(
		&b,
		"Plan: %d operation(s), %d task(s), %d config file(s), base %s\n",
		doc.OperationCount,
		doc.TaskCount,
		doc.ConfigFileCount,
		doc.Base,
	)
	for _, operation := range doc.Operations {
		fmt.Fprintf(&b, "%-7s %s", operation.Directive, operation.Target)
		if operation.Detail != "" {
			b.WriteString(operationDetailSuffix(operation))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func operationDetailSuffix(operation planOperation) string {
	switch operation.Directive {
	case "link":
		return " -> " + operation.Detail
	case "shell":
		return " [" + operation.Detail + "]"
	default:
		return " (" + operation.Detail + ")"
	}
}
