package core

import (
	"fmt"
	"os"
	"path/filepath"

	"dotbot-go/internal/expand"
)

type LinkHandler struct{}

func (LinkHandler) CanHandle(directive string) bool { return directive == "link" }
func (LinkHandler) SupportsDryRun() bool            { return true }

func (LinkHandler) Handle(ctx *Context, directive string, data any) (bool, error) {
	links, ok := asMap(data)
	if !ok {
		return false, fmt.Errorf("link directive must be a map")
	}
	defaults, _ := asMap(ctx.Defaults["link"])
	def := defaultLinkOptions(defaults)
	if def.Type != "symlink" && def.Type != "hardlink" {
		ctx.Log.Warning(fmt.Sprintf("The default link type is not recognized: '%s'", def.Type))
		return false, nil
	}
	success := true
	for linkName, target := range links {
		opts := def
		linkName = os.ExpandEnv(expand.NormSlash(linkName))
		path := ""
		if targetMap, ok := asMap(target); ok {
			opts = mergeLinkOptions(opts, targetMap)
			if opts.Type != "symlink" && opts.Type != "hardlink" {
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
			matches := createGlobResults(ctx, path, opts.Exclude)
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
