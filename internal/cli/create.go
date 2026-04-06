package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/versioning"
)

var allowedCreateTypes = []string{"added", "changed", "fixed"}

type createOptions struct {
	Bump                 string
	Type                 string
	PublicAPI            string
	Behavior             string
	Dependency           string
	Runtime              string
	Name                 string
	Body                 string
	Edit                 bool
	Breaking             bool
	Scopes               []string
	SectionKey           string
	Area                 string
	Platforms            []string
	Audiences            []string
	CustomerVisible      bool
	SupportRelevance     bool
	RequiresAction       bool
	ReleaseNotesPriority int
	DisplayOrder         int
}

func (a *App) runCreate(ctx context.Context, args []string) error {
	if wantsHelp(args) {
		a.printHelp([]string{"create"})
		return nil
	}
	a.promptIn = nil

	bumpArg, rest, err := splitCreateArgs(args)
	if err != nil {
		return err
	}

	options, err := parseCreateOptions(rest)
	if err != nil {
		return err
	}

	if a.isTTY() {
		if strings.TrimSpace(options.Name) == "" {
			options.Name, err = a.promptOptionalLine("Name stem (optional): ")
			if err != nil {
				return err
			}
		}
	}

	if err := validateCreateType(options.Type); err != nil {
		return err
	}
	if err := validateSemanticLever("public_api", options.PublicAPI, "add", "change", "remove"); err != nil {
		return err
	}
	if err := validateSemanticLever("behavior", options.Behavior, "new", "fix", "redefine"); err != nil {
		return err
	}
	if err := validateSemanticLever("dependency", options.Dependency, "refresh", "relax", "restrict"); err != nil {
		return err
	}
	if err := validateSemanticLever("runtime", options.Runtime, "expand", "reduce"); err != nil {
		return err
	}

	if options.Edit {
		if !a.isTTY() {
			return fmt.Errorf("create: --edit requires an interactive terminal")
		}
		options, err = a.editCreateOptions(bumpArg, options)
		if err != nil {
			return err
		}
	} else if strings.TrimSpace(options.Body) == "" {
		if !a.isTTY() {
			return fmt.Errorf("create: body is required; pass it as the trailing argument, use --body, or run with --edit")
		}
		options.Body, err = a.promptOptionalLine("Body (single line; use --edit for longer Markdown): ")
		if err != nil {
			return err
		}
		if strings.TrimSpace(options.Body) == "" {
			return fmt.Errorf("create: fragment body is required")
		}
	}
	if strings.TrimSpace(options.Bump) != "" {
		bumpArg = options.Bump
	}

	if err := validateCreateType(options.Type); err != nil {
		return err
	}

	repoRoot, cfg, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}

	normalizedBump, err := versioning.NormalizeBump(bumpArg)
	if err != nil {
		return err
	}

	item, err := fragments.Create(repoRoot, cfg, a.Now(), a.Random, fragments.NewInput{
		NameStem:             options.Name,
		Type:                 options.Type,
		Bump:                 normalizedBump,
		PublicAPI:            options.PublicAPI,
		Behavior:             options.Behavior,
		Dependency:           options.Dependency,
		Runtime:              options.Runtime,
		Breaking:             options.Breaking,
		Scopes:               options.Scopes,
		SectionKey:           options.SectionKey,
		Area:                 options.Area,
		Platforms:            options.Platforms,
		Audiences:            options.Audiences,
		CustomerVisible:      options.CustomerVisible,
		SupportRelevance:     options.SupportRelevance,
		RequiresAction:       options.RequiresAction,
		ReleaseNotesPriority: options.ReleaseNotesPriority,
		DisplayOrder:         options.DisplayOrder,
		Body:                 options.Body,
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.Stdout, "%s\n", item.Path)
	return nil
}

func splitCreateArgs(args []string) (string, []string, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("usage: changes create <patch|minor|major> [body] [--public-api <add|change|remove>] [--behavior <new|fix|redefine>] [--dependency <refresh|relax|restrict>] [--runtime <expand|reduce>] [--edit]")
	}

	bump := strings.TrimSpace(args[0])
	if bump == "" || strings.HasPrefix(bump, "-") {
		return "", nil, fmt.Errorf("usage: changes create <patch|minor|major> [body] [--public-api <add|change|remove>] [--behavior <new|fix|redefine>] [--dependency <refresh|relax|restrict>] [--runtime <expand|reduce>] [--edit]")
	}
	return bump, args[1:], nil
}

func parseCreateOptions(args []string) (createOptions, error) {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts createOptions
	var scopes stringSliceFlag
	var platforms stringSliceFlag
	var audiences stringSliceFlag
	var bodyFlag string

	fs.StringVar(&opts.Type, "type", "", "Optional render grouping: added, changed, or fixed")
	fs.StringVar(&opts.PublicAPI, "public-api", "", "Public API impact: add, change, or remove")
	fs.StringVar(&opts.Behavior, "behavior", "", "Behavior impact: new, fix, or redefine")
	fs.StringVar(&opts.Dependency, "dependency", "", "Dependency compatibility: refresh, relax, or restrict")
	fs.StringVar(&opts.Runtime, "runtime", "", "Runtime support: expand or reduce")
	fs.StringVar(&opts.Name, "name", "", "Optional filename stem")
	fs.StringVar(&bodyFlag, "body", "", "Fragment body")
	fs.BoolVar(&opts.Edit, "edit", false, "Open the configured editor with a scaffolded fragment")
	fs.BoolVar(&opts.Breaking, "breaking", false, "Mark entry as breaking")
	fs.Var(&scopes, "scope", "Fragment scope (repeatable)")
	fs.StringVar(&opts.SectionKey, "section-key", "", "Fragment section key")
	fs.StringVar(&opts.Area, "area", "", "Fragment product area")
	fs.Var(&platforms, "platform", "Fragment platform (repeatable)")
	fs.Var(&audiences, "audience", "Fragment audience (repeatable)")
	fs.BoolVar(&opts.CustomerVisible, "customer-visible", false, "Mark entry as customer visible")
	fs.BoolVar(&opts.SupportRelevance, "support-relevance", false, "Mark entry as support relevant")
	fs.BoolVar(&opts.RequiresAction, "requires-action", false, "Mark entry as requiring operator action")
	fs.IntVar(&opts.ReleaseNotesPriority, "release-notes-priority", 0, "Release notes priority")
	fs.IntVar(&opts.DisplayOrder, "display-order", 0, "Display order within a section")

	if err := fs.Parse(args); err != nil {
		return createOptions{}, err
	}

	opts.Scopes = scopes
	opts.Platforms = platforms
	opts.Audiences = audiences

	bodyArg := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if strings.TrimSpace(bodyFlag) != "" && bodyArg != "" {
		return createOptions{}, fmt.Errorf("create: pass the body either with --body or as the trailing argument, not both")
	}
	if strings.TrimSpace(bodyFlag) != "" {
		opts.Body = bodyFlag
	} else {
		opts.Body = bodyArg
	}

	return opts, nil
}

func validateCreateType(raw string) error {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return nil
	}
	for _, allowed := range allowedCreateTypes {
		if value == allowed {
			return nil
		}
	}
	return fmt.Errorf("create: type must be one of %s", strings.Join(allowedCreateTypes, ", "))
}

func validateSemanticLever(field, raw string, allowed ...string) error {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return nil
	}
	for _, item := range allowed {
		if value == item {
			return nil
		}
	}
	return fmt.Errorf("create: %s must be one of %s", field, strings.Join(allowed, ", "))
}

func (a *App) promptOptionalLine(label string) (string, error) {
	_, _ = fmt.Fprint(a.Stderr, label)
	reader := a.promptReader()
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

func (a *App) editCreateOptions(defaultBump string, opts createOptions) (createOptions, error) {
	path, err := a.writeCreateScaffold(defaultBump, opts)
	if err != nil {
		return createOptions{}, err
	}
	defer os.Remove(path)

	if err := a.EditFile(path); err != nil {
		return createOptions{}, err
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return createOptions{}, fmt.Errorf("read edited fragment draft: %w", err)
	}

	item, err := fragments.Parse(raw)
	if err != nil {
		return createOptions{}, fmt.Errorf("parse edited fragment draft: %w", err)
	}

	opts.Type = item.Type
	opts.PublicAPI = item.PublicAPI
	opts.Behavior = item.Behavior
	opts.Dependency = item.Dependency
	opts.Runtime = item.Runtime
	if strings.TrimSpace(item.Bump) != "" {
		defaultBump = item.Bump
	}
	if _, err := versioning.NormalizeBump(defaultBump); err != nil {
		return createOptions{}, err
	}
	opts.Bump = defaultBump
	opts.Body = item.Body
	return opts, nil
}

func (a *App) writeCreateScaffold(bump string, opts createOptions) (string, error) {
	file, err := os.CreateTemp("", "changes-create-*.md")
	if err != nil {
		return "", fmt.Errorf("create editor draft: %w", err)
	}
	defer file.Close()

	body := buildCreateScaffold(bump, opts)
	if _, err := file.WriteString(body); err != nil {
		return "", fmt.Errorf("write editor draft: %w", err)
	}
	return file.Name(), nil
}

func buildCreateScaffold(bump string, opts createOptions) string {
	var builder strings.Builder
	builder.WriteString("+++\n")
	builder.WriteString("# Required semantic version impact for this fragment today.\n")
	builder.WriteString(fmt.Sprintf("bump = %q\n", bump))
	builder.WriteString("# Optional semver reasoning levers.\n")
	builder.WriteString("# public_api = \"add|change|remove\"\n")
	if value := strings.TrimSpace(opts.PublicAPI); value != "" {
		builder.WriteString(fmt.Sprintf("public_api = %q\n", value))
	}
	builder.WriteString("# behavior = \"new|fix|redefine\"\n")
	if value := strings.TrimSpace(opts.Behavior); value != "" {
		builder.WriteString(fmt.Sprintf("behavior = %q\n", value))
	}
	builder.WriteString("# dependency = \"refresh|relax|restrict\"\n")
	if value := strings.TrimSpace(opts.Dependency); value != "" {
		builder.WriteString(fmt.Sprintf("dependency = %q\n", value))
	}
	builder.WriteString("# runtime = \"expand|reduce\"\n")
	if value := strings.TrimSpace(opts.Runtime); value != "" {
		builder.WriteString(fmt.Sprintf("runtime = %q\n", value))
	}
	builder.WriteString("# Optional render grouping for release-note sections.\n")
	builder.WriteString("# type = \"added|changed|fixed\"\n")
	if value := strings.TrimSpace(opts.Type); value != "" {
		builder.WriteString(fmt.Sprintf("type = %q\n", value))
	}
	if strings.TrimSpace(opts.Name) == "" {
		builder.WriteString("# Name stem: optional; pass --name if you want a readable filename hint.\n")
	} else {
		builder.WriteString(fmt.Sprintf("# Name stem: %s\n", opts.Name))
	}
	builder.WriteString("+++\n\n")
	return builder.String()
}

func (a *App) stdinReader() io.Reader {
	if a.Stdin != nil {
		return a.Stdin
	}
	return os.Stdin
}

func (a *App) promptReader() *bufio.Reader {
	if reader, ok := a.promptIn.(*bufio.Reader); ok {
		return reader
	}
	reader := bufio.NewReader(a.stdinReader())
	a.promptIn = reader
	return reader
}

func (a *App) isTTY() bool {
	if a.IsTTY != nil {
		return a.IsTTY()
	}
	return false
}

func (a *App) defaultIsTTY() bool {
	file, ok := a.stdinReader().(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func (a *App) defaultEditFile(path string) error {
	editor := strings.TrimSpace(os.Getenv("VISUAL"))
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if editor == "" {
		return fmt.Errorf("create: set $VISUAL or $EDITOR to use --edit")
	}

	cmd := exec.Command(editor, path)
	if stdin, ok := a.stdinReader().(*os.File); ok {
		cmd.Stdin = stdin
	}
	if stdout, ok := a.Stdout.(*os.File); ok {
		cmd.Stdout = stdout
	}
	if stderr, ok := a.Stderr.(*os.File); ok {
		cmd.Stderr = stderr
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("open editor %q: %w", editor, err)
	}
	return nil
}
