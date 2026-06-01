package core

type linkOptions struct {
	Relative      bool
	Canonicalize  bool
	Type          string
	Force         bool
	Relink        bool
	Create        bool
	Glob          bool
	Backup        bool
	Prefix        string
	If            string
	IgnoreMissing bool
	Exclude       []string
}

func defaultLinkOptions(defaults map[string]any) linkOptions {
	opts := linkOptions{Canonicalize: true, Type: "symlink"}
	if defaults == nil {
		return opts
	}
	return mergeLinkOptions(opts, defaults)
}

func mergeLinkOptions(opts linkOptions, values map[string]any) linkOptions {
	opts.Relative = boolValue(values, "relative", opts.Relative)
	if v, ok := values["canonicalize"]; ok {
		if b, ok := v.(bool); ok {
			opts.Canonicalize = b
		}
	} else {
		opts.Canonicalize = boolValue(values, "canonicalize-path", opts.Canonicalize)
	}
	opts.Type = stringValue(values, "type", opts.Type)
	opts.Force = boolValue(values, "force", opts.Force)
	opts.Relink = boolValue(values, "relink", opts.Relink)
	opts.Create = boolValue(values, "create", opts.Create)
	opts.Glob = boolValue(values, "glob", opts.Glob)
	opts.Backup = boolValue(values, "backup", opts.Backup)
	opts.Prefix = stringValue(values, "prefix", opts.Prefix)
	opts.If = stringValue(values, "if", opts.If)
	opts.IgnoreMissing = boolValue(values, "ignore-missing", opts.IgnoreMissing)
	if v, ok := values["exclude"]; ok {
		opts.Exclude = stringSlice(v)
	}
	return opts
}
