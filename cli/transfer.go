package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/coder/serpent"
	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"github.com/coder/coder/v2/cli/cliui"
	"github.com/coder/coder/v2/codersdk"
)

func (r *RootCmd) transfer() *serpent.Command {
	var (
		workspaceName string
		targetUser    string

		orgContext = NewOrganizationContext()
	)
	client := new(codersdk.Client)
	cmd := &serpent.Command{
		Annotations: workspaceCommand,
		Use:         "transfer [workspace] [target-user]",
		Short:       "Transfer a workspace",
		Long: FormatExamples(
			Example{
				Description: "Transfer a workspace to another user (if you have permission)",
				Command:     "coder transfer <workspace_name> <target-username>",
			},
		),
		Middleware: serpent.Chain(r.InitClient(client)),
		Handler: func(inv *serpent.Invocation) error {
			// Step 1: call API to initiate transfer, receive build ID
			// Step 2: poll build ID, show logs
			// Step 3: once complete, trigger start transition

			var (
				err error

				user codersdk.User
			)
			workspaceOwner := codersdk.Me
			if len(inv.Args) >= 1 {
				workspaceOwner, workspaceName, err = splitNamedWorkspace(inv.Args[0])
				if err != nil {
					return err
				}
			}
			if len(inv.Args) >= 2 {
				targetUser = inv.Args[1]
			}

			if workspaceName == "" {
				workspaceName, err = cliui.Prompt(inv, cliui.PromptOptions{
					Text: "Specify the name of the workspace to transfer:",
					Validate: func(workspaceName string) error {
						err = codersdk.NameValid(workspaceName)
						if err != nil {
							return xerrors.Errorf("workspace name %q is invalid: %w", workspaceName, err)
						}

						return nil
					},
				})
				if err != nil {
					return xerrors.Errorf("workspace %q is invalid: %w", workspaceName, err)
				}
			}

			// TODO: check authorization
			workspace, err := client.WorkspaceByOwnerAndName(inv.Context(), workspaceOwner, workspaceName, codersdk.WorkspaceOptions{})
			if err != nil {
				return xerrors.Errorf("cannot find workspace %q: %w", workspaceName, err)
			}

			if targetUser == "" {
				targetUser, err = cliui.Prompt(inv, cliui.PromptOptions{
					Text: "Specify the username to transfer the workspace to:",
					Validate: func(s string) error {
						// Happens further down.
						return nil
					},
				})
				if err != nil {
					return xerrors.Errorf("user %q is invalid: %w", targetUser, err)
				}
			}

			err = codersdk.NameValid(targetUser)
			if err != nil {
				return xerrors.Errorf("target user %q is invalid: %w", targetUser, err)
			}

			// TODO: check authorization
			user, err = client.User(inv.Context(), targetUser)
			if err != nil {
				return err
			}

			if workspace.OwnerID == user.ID {
				return xerrors.Errorf("%q already owns %q", user.Name, workspaceName)
			}

			fmt.Fprintln(inv.Stdout, workspace.OwnerID, user.ID)
			inv.Stdout.Write([]byte(fmt.Sprintf("%s:%s to %s\n", workspaceOwner, workspaceName, user.Username)))

			// TODO: return WorkspaceBuild entity
			buildID, err := client.TransferWorkspace(inv.Context(), workspaceOwner, workspace.Name, user.ID)
			if err != nil {
				return xerrors.Errorf("transfer failed: %w", err)
			}

			err = waitForBuild(inv.Context(), inv.Stdout, client, buildID)
			if err != nil {
				return xerrors.Errorf("post-transfer stop failed: %w", err)
			}

			wb, err := client.CreateWorkspaceBuild(inv.Context(), workspace.ID, codersdk.CreateWorkspaceBuildRequest{
				TemplateVersionID: workspace.TemplateActiveVersionID,
				Transition:        codersdk.WorkspaceTransitionStart,
			})
			if err != nil {
				return xerrors.Errorf("transfer failed: %w", err)
			}

			err = waitForBuild(inv.Context(), inv.Stdout, client, wb.ID)
			if err != nil {
				return xerrors.Errorf("post-transfer start failed: %w", err)
			}

			return nil
		},
	}
	// cmd.Options = append(cmd.Options,
	// 	serpent.Option{
	// 		Flag:          "to",
	// 		FlagShorthand: "u",
	// 		Env:           "CODER_TRANSFER_TARGET_USER",
	// 		Description:   "Specify a user to transfer the workspace to.",
	// 		Value:         serpent.StringOf(&targetUser),
	// 	},
	// 	cliui.SkipPromptOption(),
	// )
	orgContext.AttachOptions(cmd)
	return cmd
}

// Copied from scaletest/workspacebuild/run.go
func waitForBuild(ctx context.Context, w io.Writer, client *codersdk.Client, buildID uuid.UUID) error {
	_, _ = fmt.Fprint(w, "Build is currently queued...")

	// Wait for build to start.
	for {
		build, err := client.WorkspaceBuild(ctx, buildID)
		if err != nil {
			return xerrors.Errorf("fetch build: %w", err)
		}

		if build.Job.Status != codersdk.ProvisionerJobPending {
			break
		}

		_, _ = fmt.Fprint(w, ".")
		time.Sleep(500 * time.Millisecond)
	}

	_, _ = fmt.Fprintln(w, "\nBuild started! Streaming logs below:")

	logs, closer, err := client.WorkspaceBuildLogsAfter(ctx, buildID, 0)
	if err != nil {
		return xerrors.Errorf("start streaming build logs: %w", err)
	}
	defer closer.Close()

	currentStage := ""
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case log, ok := <-logs:
			if !ok {
				build, err := client.WorkspaceBuild(ctx, buildID)
				if err != nil {
					return xerrors.Errorf("fetch build: %w", err)
				}

				_, _ = fmt.Fprintln(w, "")
				switch build.Job.Status {
				case codersdk.ProvisionerJobSucceeded:
					_, _ = fmt.Fprintln(w, "\nBuild succeeded!")
					return nil
				case codersdk.ProvisionerJobFailed:
					_, _ = fmt.Fprintf(w, "\nBuild failed with error %q.\nSee logs above for more details.\n", build.Job.Error)
					return xerrors.Errorf("build failed with status %q: %s", build.Job.Status, build.Job.Error)
				case codersdk.ProvisionerJobCanceled:
					_, _ = fmt.Fprintln(w, "\nBuild canceled.")
					return xerrors.New("build canceled")
				default:
					_, _ = fmt.Fprintf(w, "\nLogs disconnected with unexpected job status %q and error %q.\n", build.Job.Status, build.Job.Error)
					return xerrors.Errorf("logs disconnected with unexpected job status %q and error %q", build.Job.Status, build.Job.Error)
				}
			}

			if log.Stage != currentStage {
				currentStage = log.Stage
				_, _ = fmt.Fprintf(w, "\n%s\n", currentStage)
			}

			level := "unknown"
			if log.Level != "" {
				level = string(log.Level)
			}
			_, _ = fmt.Fprintf(w, "\t%s:\t%s\n", level, log.Output)
		}
	}
}
