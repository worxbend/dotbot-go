package core

import (
	"fmt"
	"os"
	"path/filepath"

	"dotbot-go/internal/expand"
)

// LinkHandler implements the link directive.
type LinkHandler struct{}

// CanHandle reports whether directive is link.
func (LinkHandler) CanHandle(directive string) bool { return directive == "link" }

// SupportsDryRun reports that link can preview link creation and replacement.
func (LinkHandler) SupportsDryRun() bool { return true }

// Validate checks link directive data without touching the filesystem.
func (LinkHandler) Validate(ctx *Context, directive string, data any) error {
	links, ok := asMap(data)
	if !ok {
		return fmt.Errorf("link directive must be a map")
	}
	defaults, _ := asMap(ctx.Defaults["link"])
	def := defaultLinkOptions(defaults)
	if !validLinkType(def.Type) {
		return fmt.Errorf("default link type is not recognized: %s", def.Type)
	}
	for _, linkName := range sortedKeys(links) {
		target := links[linkName]
		opts := def
		if targetMap, ok := asMap(target); ok {
			if err := validateLinkMap(linkName, targetMap); err != nil {
				return err
			}
			opts = mergeLinkOptions(opts, targetMap)
			if !validLinkType(opts.Type) {
				return fmt.Errorf("link type is not recognized: %s", opts.Type)
			}
			continue
		}
		if target != nil {
			if _, ok := asString(target); !ok {
				return fmt.Errorf("link target for %s must be a string or map", linkName)
			}
		}
	}
	return nil
}

// Plan expands link directive data into link operations.
func (h LinkHandler) Plan(ctx *Context, directive string, data any) ([]Operation, error) {
	if err := h.Validate(ctx, directive, data); err != nil {
		return nil, err
	}
	links, _ := asMap(data)
	operations := []Operation{}
	for _, linkName := range sortedKeys(links) {
		target := links[linkName]
		detail := ""
		if targetMap, ok := asMap(target); ok {
			detail = defaultTarget(linkName, targetMap["path"])
		} else {
			detail = defaultTarget(linkName, target)
		}
		operations = append(operations, Operation{
			Directive: directive,
			Target:    linkName,
			Detail:    detail,
		})
	}
	return operations, nil
}

// Handle creates or updates links requested by the link directive.
func (LinkHandler) Handle(ctx *Context, directive string, data any) (bool, error) {
	links, ok := asMap(data)
	if !ok {
		return false, fmt.Errorf("link directive must be a map")
	}
	defaults, _ := asMap(ctx.Defaults["link"])
	def := defaultLinkOptions(defaults)
	if !validLinkType(def.Type) {
		ctx.Log.Warning(fmt.Sprintf("The default link type is not recognized: '%s'", def.Type))
		return false, nil
	}
	success := true
	for _, linkName := range sortedKeys(links) {
		target := links[linkName]
		opts := def
		linkName = os.ExpandEnv(expand.NormSlash(linkName))
		path := ""
		if targetMap, ok := asMap(target); ok {
			opts = mergeLinkOptions(opts, targetMap)
			if !validLinkType(opts.Type) {
				ctx.Log.Warning(fmt.Sprintf("The link type is not recognized: '%s'", opts.Type))
				success = false
				continue
			}
			path = defaultTarget(linkName, targetMap["path"])
		} else {
			path = defaultTarget(linkName, target)
		}
		path = filepath.Clean(expand.Path(path))
		if opts.If != "" && !testSuccess(ctx, opts.If) {
			ctx.Log.Info(fmt.Sprintf("Skipping %s", linkName))
			continue
		}
		if opts.Glob && hasGlobChars(path) {
			matches, err := createGlobResults(path, opts.Exclude)
			if err != nil {
				ctx.Log.Warning(fmt.Sprintf("Unable to expand glob '%s': %v", path, err))
				success = false
				continue
			}
			ctx.Log.Debug(fmt.Sprintf("Globs from '%s': %v", path, matches))
			for _, fullItem := range matches {
				globItem := globLinkItem(path, fullItem)
				if opts.Prefix != "" {
					globItem = opts.Prefix + globItem
				}
				globLinkName := filepath.Join(linkName, globItem)
				success = processOneLink(ctx, fullItem, globLinkName, opts, true) && success
			}
			continue
		}
		success = processOneLink(ctx, path, linkName, opts, false) && success
	}
	return finish(ctx, success, "All links have been set up", "Some links were not successfully set up"), nil
}

func validateLinkMap(linkName string, values map[string]any) error {
	if path, ok := values["path"]; ok && path != nil {
		if _, ok := asString(path); !ok {
			return fmt.Errorf("link path for %s must be a string", linkName)
		}
	}
	if typ, ok := values["type"]; ok {
		if _, ok := asString(typ); !ok {
			return fmt.Errorf("link type for %s must be a string", linkName)
		}
	}
	if exclude, ok := values["exclude"]; ok {
		if !isStringList(exclude) {
			return fmt.Errorf("link exclude for %s must be a list of strings", linkName)
		}
	}
	return nil
}

func validLinkType(value string) bool {
	return value == "symlink" || value == "hardlink"
}
