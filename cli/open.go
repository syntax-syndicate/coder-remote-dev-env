package cli

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/skratchdot/open-golang/open"
	"golang.org/x/xerrors"

	"github.com/coder/coder/v2/cli/cliui"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/serpent"
)

func (r *RootCmd) open() *serpent.Command {
	cmd := &serpent.Command{
		Use:   "open",
		Short: "Open a workspace",
		Handler: func(inv *serpent.Invocation) error {
			return inv.Command.HelpHandler(inv)
		},
		Children: []*serpent.Command{
			r.openVSCode(),
			r.openFleet(),
			r.openCursor(),
			r.openZed(),
			r.openWindsurf(),
		},
	}
	return cmd
}

type ideConfig struct {
	scheme        string
	host          string
	path          string
	displayName   string
	dirParamName  string
	generateToken bool
	testOpenError bool
}

const (
	vscodeDesktopName = "VS Code Desktop"
	// IDE specific protocol handlers
	vsCodeScheme   = "vscode"
	fleetScheme    = "fleet"
	cursorScheme   = "cursor"
	zedScheme      = "zed"
	windsurfScheme = "windsurf"
)

func (r *RootCmd) openVSCode() *serpent.Command {
	var (
		generateToken    bool
		testOpenError    bool
		appearanceConfig codersdk.AppearanceConfig
	)

	client := new(codersdk.Client)
	cmd := &serpent.Command{
		Annotations: workspaceCommand,
		Use:         "vscode <workspace> [<directory in workspace>]",
		Short:       fmt.Sprintf("Open a workspace in %s", vscodeDesktopName),
		Middleware: serpent.Chain(
			serpent.RequireRangeArgs(1, 2),
			r.InitClient(client),
			initAppearance(client, &appearanceConfig),
		),
		Handler: func(inv *serpent.Invocation) error {
			config := ideConfig{
				scheme:        vsCodeScheme,
				host:          "coder.coder-remote",
				path:          "/open",
				displayName:   vscodeDesktopName,
				dirParamName:  "folder",
				generateToken: generateToken,
				testOpenError: testOpenError,
			}
			
			return r.openIDEHandler(inv, client, config, appearanceConfig)
		},
	}

	cmd.Options = serpent.OptionSet{
		{
			Flag: "generate-token",
			Env:  "CODER_OPEN_VSCODE_GENERATE_TOKEN",
			Description: fmt.Sprintf(
				"Generate an auth token and include it in the vscode:// URI. This is for automagical configuration of %s and not needed if already configured. "+
					"This flag does not need to be specified when running this command on a local machine unless automatic open fails.",
				vscodeDesktopName,
			),
			Value: serpent.BoolOf(&generateToken),
		},
		{
			Flag:        "test.open-error",
			Description: "Don't run the open command.",
			Value:       serpent.BoolOf(&testOpenError),
			Hidden:      true, // This is for testing\!
		},
	}

	return cmd
}

// waitForAgentCond uses the watch workspace API to update the agent information
// until the condition is met.
func waitForAgentCond(ctx context.Context, client *codersdk.Client, workspace codersdk.Workspace, workspaceAgent codersdk.WorkspaceAgent, cond func(codersdk.WorkspaceAgent) bool) (codersdk.Workspace, codersdk.WorkspaceAgent, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if cond(workspaceAgent) {
		return workspace, workspaceAgent, nil
	}

	wc, err := client.WatchWorkspace(ctx, workspace.ID)
	if err \!= nil {
		return workspace, workspaceAgent, xerrors.Errorf("watch workspace: %w", err)
	}

	for workspace = range wc {
		workspaceAgent, err = getWorkspaceAgent(workspace, workspaceAgent.Name)
		if err \!= nil {
			return workspace, workspaceAgent, xerrors.Errorf("get workspace agent: %w", err)
		}
		if cond(workspaceAgent) {
			return workspace, workspaceAgent, nil
		}
	}

	return workspace, workspaceAgent, xerrors.New("watch workspace: unexpected closed channel")
}

// isWindowsAbsPath does a simplistic check for if the path is an absolute path
// on Windows. Drive letter or preceding `\` is interpreted as absolute.
func isWindowsAbsPath(p string) bool {
	// Remove the drive letter, if present.
	if len(p) >= 2 && p[1] == ':' {
		p = p[2:]
	}

	switch {
	case len(p) == 0:
		return false
	case p[0] == '\\':
		return true
	default:
		return false
	}
}

// windowsJoinPath joins the elements into a path, using Windows path separator
// and converting forward slashes to backslashes.
func windowsJoinPath(elem ...string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(elem...)
	}

	var s string
	for _, e := range elem {
		e = unixToWindowsPath(e)
		if e == "" {
			continue
		}
		if s == "" {
			s = e
			continue
		}
		s += "\\" + strings.TrimSuffix(e, "\\")
	}
	return s
}

func unixToWindowsPath(p string) string {
	return strings.ReplaceAll(p, "/", "\\")
}

// resolveAgentAbsPath resolves the absolute path to a file or directory in the
// workspace. If the path is relative, it will be resolved relative to the
// workspace's expanded directory. If the path is absolute, it will be returned
// as-is. If the path is relative and the workspace directory is not expanded,
// an error will be returned.
//
// If the path is being resolved within the workspace, the path will be resolved
// relative to the current working directory.
func resolveAgentAbsPath(workingDirectory, relOrAbsPath, agentOS string, local bool) (string, error) {
	switch {
	case relOrAbsPath == "":
		return workingDirectory, nil

	case relOrAbsPath == "~" || strings.HasPrefix(relOrAbsPath, "~/"):
		return "", xerrors.Errorf("path %q requires expansion and is not supported, use an absolute path instead", relOrAbsPath)

	case local:
		p, err := filepath.Abs(relOrAbsPath)
		if err \!= nil {
			return "", xerrors.Errorf("expand path: %w", err)
		}
		return p, nil

	case agentOS == "windows":
		relOrAbsPath = unixToWindowsPath(relOrAbsPath)
		switch {
		case workingDirectory \!= "" && \!isWindowsAbsPath(relOrAbsPath):
			return windowsJoinPath(workingDirectory, relOrAbsPath), nil
		case isWindowsAbsPath(relOrAbsPath):
			return relOrAbsPath, nil
		default:
			return "", xerrors.Errorf("path %q not supported, use an absolute path instead", relOrAbsPath)
		}

	// Note that we use `path` instead of `filepath` since we want Unix behavior.
	case workingDirectory \!= "" && \!path.IsAbs(relOrAbsPath):
		return path.Join(workingDirectory, relOrAbsPath), nil
	case path.IsAbs(relOrAbsPath):
		return relOrAbsPath, nil
	default:
		return "", xerrors.Errorf("path %q not supported, use an absolute path instead", relOrAbsPath)
	}
}


func doAsync(f func()) (wait func()) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()
	return func() {
		<-done
	}
}

// openIDEHandler creates a handler function for opening workspaces in various IDEs
func (r *RootCmd) openIDEHandler(inv *serpent.Invocation, client *codersdk.Client, config ideConfig, appearanceConfig codersdk.AppearanceConfig) error {
	ctx, cancel := context.WithCancel(inv.Context())
	defer cancel()

	// Check if we're inside a workspace
	insideAWorkspace := inv.Environ.Get("CODER") == "true"
	inWorkspaceName := inv.Environ.Get("CODER_WORKSPACE_NAME") + "." + inv.Environ.Get("CODER_WORKSPACE_AGENT_NAME")

	// Get workspace and agent
	workspaceQuery := inv.Args[0]
	autostart := true
	workspace, workspaceAgent, err := getWorkspaceAndAgent(ctx, inv, client, autostart, workspaceQuery)
	if err \!= nil {
		return xerrors.Errorf("get workspace and agent: %w", err)
	}

	workspaceName := workspace.Name + "." + workspaceAgent.Name
	insideThisWorkspace := insideAWorkspace && inWorkspaceName == workspaceName

	if \!insideThisWorkspace {
		// Wait for the agent to connect
		err = cliui.Agent(ctx, inv.Stderr, workspaceAgent.ID, cliui.AgentOptions{
			Fetch:     client.WorkspaceAgent,
			FetchLogs: nil,
			Wait:      false,
			DocsURL:   appearanceConfig.DocsURL,
		})
		if err \!= nil {
			if xerrors.Is(err, context.Canceled) {
				return cliui.Canceled
			}
			return xerrors.Errorf("agent: %w", err)
		}

		if workspaceAgent.Directory \!= "" {
			workspace, workspaceAgent, err = waitForAgentCond(ctx, client, workspace, workspaceAgent, func(a codersdk.WorkspaceAgent) bool {
				return workspaceAgent.LifecycleState \!= codersdk.WorkspaceAgentLifecycleCreated
			})
			if err \!= nil {
				return xerrors.Errorf("wait for agent: %w", err)
			}
		}
	}

	// Parse the directory argument if provided
	var directory string
	if len(inv.Args) > 1 {
		directory = inv.Args[1]
	}
	directory, err = resolveAgentAbsPath(workspaceAgent.ExpandedDirectory, directory, workspaceAgent.OperatingSystem, insideThisWorkspace)
	if err \!= nil {
		return xerrors.Errorf("resolve agent path: %w", err)
	}

	// Generate the appropriate URL based on the scheme
	var u *url.URL
	
	// Use different URL formats depending on the scheme
	switch config.scheme {
	case vsCodeScheme, cursorScheme, windsurfScheme:
		// VSCode, Cursor, and Windsurf use similar URL formats
		u = &url.URL{
			Scheme: config.scheme,
			Host:   config.host \!= "" ? config.host : "coder.coder-remote",
			Path:   config.path \!= "" ? config.path : "/open",
		}
		
		qp := url.Values{}
		qp.Add("url", client.URL.String())
		qp.Add("owner", workspace.OwnerName)
		qp.Add("workspace", workspace.Name)
		qp.Add("agent", workspaceAgent.Name)
		
		// Add directory if present - use the appropriate parameter name
		dirParam := config.dirParamName
		if dirParam == "" {
			dirParam = "folder" // Default to folder for VSCode-like schemes
		}
		
		if directory \!= "" {
			qp.Add(dirParam, directory)
		}
		
		// Add token if needed
		if \!insideAWorkspace || config.generateToken {
			apiKey, err := client.CreateAPIKey(ctx, codersdk.Me)
			if err \!= nil {
				return xerrors.Errorf("create API key: %w", err)
			}
			qp.Add("token", apiKey.Key)
		}
		
		u.RawQuery = qp.Encode()
		
	case fleetScheme:
		// Fleet uses a specific URI format: fleet://fleet.ssh/ssh://username@hostname:22
		username := "coder"
		hostname := client.URL.Hostname()
		
		u = &url.URL{
			Scheme: fleetScheme,
			Host:   "fleet.ssh",
			Path:   fmt.Sprintf("/ssh://%s@%s:22", username, hostname),
		}
		// Fleet doesn't support query parameters, so we don't add them
		
	case zedScheme:
		// Zed uses a simpler format per PR #19970
		// Format: zed://coder@hostname:22
		// Reference: https://github.com/zed-industries/zed/pull/19970/files
		username := "coder"
		hostname := client.URL.Hostname()
		
		u = &url.URL{
			Scheme: zedScheme,
			Path:   fmt.Sprintf("//%s@%s:22", username, hostname),
		}
		// Zed doesn't support query parameters, so we don't add them
	}

	openingPath := workspaceName
	if directory \!= "" {
		openingPath += ":" + directory
	}

	if insideAWorkspace {
		_, _ = fmt.Fprintf(inv.Stderr, "Opening %s in %s is not supported inside a workspace, please open the following URI on your local machine instead:\n\n", openingPath, config.displayName)
		_, _ = fmt.Fprintf(inv.Stdout, "%s\n", u.String())
		return nil
	}
	_, _ = fmt.Fprintf(inv.Stderr, "Opening %s in %s\n", openingPath, config.displayName)

	if \!config.testOpenError {
		err = open.Run(u.String())
	} else {
		err = xerrors.New("test.open-error")
	}
	
	if err \!= nil {
		// If token was generated, try to clean it up
		if u.Query().Get("token") \!= "" {
			token := u.Query().Get("token")
			wait := doAsync(func() {
				// Best effort, we don't care if this fails.
				apiKeyID := strings.SplitN(token, "-", 2)[0]
				_ = client.DeleteAPIKey(ctx, codersdk.Me, apiKeyID)
			})
			defer wait()
		}

		_, _ = fmt.Fprintf(inv.Stderr, "Could not automatically open %s in %s: %s\n", openingPath, config.displayName, err)
		_, _ = fmt.Fprintf(inv.Stderr, "Please open the following URI instead:\n\n")
		_, _ = fmt.Fprintf(inv.Stdout, "%s\n", u.String())
		return nil
	}

	return nil
}

// openWindsurf provides a direct command for opening workspaces in Windsurf
func (r *RootCmd) openWindsurf() *serpent.Command {
	var (
		generateToken    bool
		testOpenError    bool
		appearanceConfig codersdk.AppearanceConfig
	)

	client := new(codersdk.Client)
	cmd := &serpent.Command{
		Annotations: workspaceCommand,
		Use:         "windsurf <workspace> [<directory in workspace>]",
		Short:       "Open a workspace in Windsurf",
		Middleware: serpent.Chain(
			serpent.RequireRangeArgs(1, 2),
			r.InitClient(client),
			initAppearance(client, &appearanceConfig),
		),
		Handler: func(inv *serpent.Invocation) error {
			config := ideConfig{
				scheme:        windsurfScheme,
				host:          "coder.coder-remote", // VSCode-like URL format
				path:          "/open",
				displayName:   "Windsurf IDE",
				dirParamName:  "folder", // Using same param name as VSCode
				generateToken: generateToken,
				testOpenError: testOpenError,
			}
			
			return r.openIDEHandler(inv, client, config, appearanceConfig)
		},
	}

	cmd.Options = serpent.OptionSet{
		{
			Flag: "generate-token",
			Env:  "CODER_OPEN_WINDSURF_GENERATE_TOKEN",
			Description: "Generate an auth token and include it in the Windsurf URI. This is for automagical configuration of Windsurf and not needed if already configured. " +
				"This flag does not need to be specified when running this command on a local machine unless automatic open fails.",
			Value: serpent.BoolOf(&generateToken),
		},
		{
			Flag:        "test.open-error",
			Description: "Don't run the open command.",
			Value:       serpent.BoolOf(&testOpenError),
			Hidden:      true, // This is for testing\!
		},
	}

	return cmd
}

// openFleet provides a direct command for opening workspaces in JetBrains Fleet
func (r *RootCmd) openFleet() *serpent.Command {
	var (
		generateToken    bool
		testOpenError    bool
		appearanceConfig codersdk.AppearanceConfig
	)

	client := new(codersdk.Client)
	cmd := &serpent.Command{
		Annotations: workspaceCommand,
		Use:         "fleet <workspace> [<directory in workspace>]",
		Short:       "Open a workspace in JetBrains Fleet",
		Middleware: serpent.Chain(
			serpent.RequireRangeArgs(1, 2),
			r.InitClient(client),
			initAppearance(client, &appearanceConfig),
		),
		Handler: func(inv *serpent.Invocation) error {
			config := ideConfig{
				scheme:        fleetScheme,
				displayName:   "JetBrains Fleet",
				dirParamName:  "dir",
				generateToken: generateToken,
				testOpenError: testOpenError,
			}
			
			return r.openIDEHandler(inv, client, config, appearanceConfig)
		},
	}

	cmd.Options = serpent.OptionSet{
		{
			Flag: "generate-token",
			Env:  "CODER_OPEN_FLEET_GENERATE_TOKEN",
			Description: "Generate an auth token and include it in the Fleet URI. This is for automagical configuration of Fleet and not needed if already configured. " +
				"This flag does not need to be specified when running this command on a local machine unless automatic open fails.",
			Value: serpent.BoolOf(&generateToken),
		},
		{
			Flag:        "test.open-error",
			Description: "Don't run the open command.",
			Value:       serpent.BoolOf(&testOpenError),
			Hidden:      true, // This is for testing\!
		},
	}

	return cmd
}

// openCursor provides a direct command for opening workspaces in Cursor
func (r *RootCmd) openCursor() *serpent.Command {
	var (
		generateToken    bool
		testOpenError    bool
		appearanceConfig codersdk.AppearanceConfig
	)

	client := new(codersdk.Client)
	cmd := &serpent.Command{
		Annotations: workspaceCommand,
		Use:         "cursor <workspace> [<directory in workspace>]",
		Short:       "Open a workspace in Cursor",
		Middleware: serpent.Chain(
			serpent.RequireRangeArgs(1, 2),
			r.InitClient(client),
			initAppearance(client, &appearanceConfig),
		),
		Handler: func(inv *serpent.Invocation) error {
			config := ideConfig{
				scheme:        cursorScheme,
				host:          "coder.coder-remote", // VSCode-like URL format
				path:          "/open",
				displayName:   "Cursor",
				dirParamName:  "folder", // Using same param name as VSCode
				generateToken: generateToken,
				testOpenError: testOpenError,
			}
			
			return r.openIDEHandler(inv, client, config, appearanceConfig)
		},
	}

	cmd.Options = serpent.OptionSet{
		{
			Flag: "generate-token",
			Env:  "CODER_OPEN_CURSOR_GENERATE_TOKEN",
			Description: "Generate an auth token and include it in the Cursor URI. This is for automagical configuration of Cursor and not needed if already configured. " +
				"This flag does not need to be specified when running this command on a local machine unless automatic open fails.",
			Value: serpent.BoolOf(&generateToken),
		},
		{
			Flag:        "test.open-error",
			Description: "Don't run the open command.",
			Value:       serpent.BoolOf(&testOpenError),
			Hidden:      true, // This is for testing\!
		},
	}

	return cmd
}

// openZed provides a direct command for opening workspaces in Zed Editor
func (r *RootCmd) openZed() *serpent.Command {
	var (
		generateToken    bool
		testOpenError    bool
		appearanceConfig codersdk.AppearanceConfig
	)

	client := new(codersdk.Client)
	cmd := &serpent.Command{
		Annotations: workspaceCommand,
		Use:         "zed <workspace> [<directory in workspace>]",
		Short:       "Open a workspace in Zed Editor",
		Middleware: serpent.Chain(
			serpent.RequireRangeArgs(1, 2),
			r.InitClient(client),
			initAppearance(client, &appearanceConfig),
		),
		Handler: func(inv *serpent.Invocation) error {
			config := ideConfig{
				scheme:        zedScheme,
				displayName:   "Zed Editor",
				dirParamName:  "dir",
				generateToken: generateToken,
				testOpenError: testOpenError,
			}
			
			return r.openIDEHandler(inv, client, config, appearanceConfig)
		},
	}

	cmd.Options = serpent.OptionSet{
		{
			Flag: "generate-token",
			Env:  "CODER_OPEN_ZED_GENERATE_TOKEN",
			Description: "Generate an auth token and include it in the Zed URI. This is for automagical configuration of Zed and not needed if already configured. " +
				"This flag does not need to be specified when running this command on a local machine unless automatic open fails.",
			Value: serpent.BoolOf(&generateToken),
		},
		{
			Flag:        "test.open-error",
			Description: "Don't run the open command.",
			Value:       serpent.BoolOf(&testOpenError),
			Hidden:      true, // This is for testing\!
		},
	}

	return cmd
}
