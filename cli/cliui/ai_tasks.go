package cliui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jedib0t/go-pretty/v6/table"
	"golang.org/x/xerrors"

	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/pretty"
	"github.com/coder/serpent"
)

func AITasks(inv *serpent.Invocation, client *codersdk.Client) error {
	initModel := initialModel(client, inv.Context())

	p := tea.NewProgram(
		initModel,
		tea.WithContext(inv.Context()),
		tea.WithInput(inv.Stdin),
		tea.WithOutput(inv.Stdout),
	)

	m, err := p.Run()
	if err != nil {
		return err
	}

	model, ok := m.(aiTasksModel)
	if !ok {
		return xerrors.New(fmt.Sprintf("unknown model found %T (%+v)", m, m))
	}

	if model.canceled {
		return Canceled
	}

	return nil
}

type aiTasksModel struct {
	ctx          context.Context
	client       *codersdk.Client
	canceled     bool
	tasks        []aiTask
	inputs       []textinput.Model
	activeInput  int
	focusedInput bool
}

func initialModel(client *codersdk.Client, ctx context.Context) aiTasksModel {
	return aiTasksModel{
		client:       client,
		tasks:        []aiTask{},
		canceled:     false,
		ctx:          ctx,
		inputs:       []textinput.Model{},
		activeInput:  0,
		focusedInput: false,
	}
}

type aiTask struct {
	summary        string
	waitingOnInput bool
	workspace      codersdk.Workspace
}

func fetchTasks(ctx context.Context, client *codersdk.Client) ([]aiTask, error) {

	workspaces, err := client.Workspaces(ctx, codersdk.WorkspaceFilter{Owner: "me"})
	if err != nil {
		// TODO: make return error cmd
		return nil, xerrors.Errorf("could not fetch owned workspaces: %w", err)
	}

	tasks := []aiTask{}
	for _, w := range workspaces.Workspaces {
		for _, r := range w.LatestBuild.Resources {
			for _, a := range r.Agents {
				if len(a.Tasks) != 0 {
					mostRecentTask := a.Tasks[0]
					tasks = append(tasks, aiTask{
						summary:        mostRecentTask.Summary,
						waitingOnInput: a.TaskWaitingForUserInput,
						workspace:      w,
					})
				}
			}
		}

	}

	return tasks, nil
}

type newTasksFetched struct {
	tasks []aiTask
}

type WorkspaceAgentIOConnection struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func getWorkspaceAgentIoConnection(workspaceName string) (*WorkspaceAgentIOConnection, error) {
	cmd := exec.Command("coder", "ssh", workspaceName)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, xerrors.Errorf("failed to get stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, xerrors.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, xerrors.Errorf("failed to start command: %w", err)
	}

	return &WorkspaceAgentIOConnection{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
	}, nil
}

func (wc *WorkspaceAgentIOConnection) Close() error {
	// Should close the stdin/stdout pipes
	if err := wc.cmd.Wait(); err != nil {
		return err
	}

	return nil
}

func (wc *WorkspaceAgentIOConnection) Write(p []byte) (int, error) {
	return wc.stdin.Write(p)
}

func (wc *WorkspaceAgentIOConnection) Read(p []byte) (int, error) {
	return wc.stdout.Read(p)
}

func (m aiTasksModel) Init() tea.Cmd {
	return func() tea.Msg {
		tasks, err := fetchTasks(m.ctx, m.client)
		if err != nil {
			return err
		}
		return newTasksFetched{tasks: tasks}
	}
}

func (m aiTasksModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.canceled = true
			return m, tea.Quit
		case "tab":
			// Switch focus between inputs
			if len(m.inputs) > 0 {
				if m.focusedInput {
					m.inputs[m.activeInput].Blur()
					m.activeInput = (m.activeInput + 1) % len(m.inputs)
					m.inputs[m.activeInput].Focus()
					return m, nil
				} else {
					m.focusedInput = true
					m.inputs[m.activeInput].Focus()
					return m, nil
				}
			}
		case "shift+tab":
			// Switch focus between inputs in reverse
			if len(m.inputs) > 0 && m.focusedInput {
				m.inputs[m.activeInput].Blur()
				m.activeInput = (m.activeInput - 1 + len(m.inputs)) % len(m.inputs)
				m.inputs[m.activeInput].Focus()
				return m, nil
			}
		case "enter":
			// Process the input when Enter is pressed
			if m.focusedInput && len(m.inputs) > 0 && m.activeInput < len(m.tasks) {
				currentInput := m.inputs[m.activeInput].Value()
				if currentInput != "" {
					// Communicate with the running Claude session over stdin/stdout
					task := m.tasks[m.activeInput]
					workspaceName := task.workspace.Name

					// Create a command to execute the needed task
					pipes := tea.Batch(
						func() tea.Msg {

							// Read output from the command
							output := &bytes.Buffer{}
							_, err = io.Copy(output, stdout)
							if err != nil {
								return xerrors.Errorf("failed to read output: %w", err)
							}

							if err := cmd.Wait(); err != nil {
								return xerrors.Errorf("command failed: %w", err)
							}

							// Process the output if needed
							log.Printf("Output from Claude: %s", output.String())

							// Fetch tasks again to update status
							tasks, err := fetchTasks(m.ctx, m.client)
							if err != nil {
								return err
							}
							return newTasksFetched{tasks: tasks}
						},
					)

					// Reset the input field
					m.inputs[m.activeInput].SetValue("")
					return m, pipes
				}
			}
		}

		// Handle input changes for the focused input
		if m.focusedInput && len(m.inputs) > 0 {
			var cmd tea.Cmd
			m.inputs[m.activeInput], cmd = m.inputs[m.activeInput].Update(msg)
			return m, cmd
		}

	case terminateMsg:
		m.canceled = true
		return m, tea.Quit

	case newTasksFetched:
		m.tasks = msg.tasks

		// Initialize text inputs for each workspace
		m.inputs = make([]textinput.Model, len(m.tasks))
		for i, task := range m.tasks {
			ti := textinput.New()
			ti.Placeholder = "Type your response here..."
			ti.CharLimit = 500
			ti.Width = 50
			ti.Prompt = ""
			if task.waitingOnInput {
				ti.Prompt = "> "
			}
			m.inputs[i] = ti
		}

		return m, nil
	}

	return m, cmd
}

func (m aiTasksModel) View() string {
	if len(m.tasks) == 0 {
		return "No AI tasks found."
	}

	var output string

	tableWriter := table.NewWriter()
	tableWriter.SetStyle(table.StyleLight)
	tableWriter.Style().Options.SeparateColumns = false
	tableWriter.AppendHeader(table.Row{"Workspace", "Task Summary", "Status", "Link"})

	for _, task := range m.tasks {
		status := "Working"
		if task.waitingOnInput {
			status = pretty.Sprint(DefaultStyles.Warn, "Waiting for input")
		}

		tableWriter.AppendRow(table.Row{
			task.workspace.Name,
			task.summary,
			status,
			pretty.Sprint(DefaultStyles.Code, "coder ssh "+task.workspace.Name),
		})
	}

	output = tableWriter.Render()
	output += "\n\n"

	// Add text inputs only for workspaces waiting for input
	for i, task := range m.tasks {
		if len(m.inputs) > i && task.waitingOnInput {
			workspaceLabel := task.workspace.Name
			if i == m.activeInput && m.focusedInput {
				workspaceLabel = pretty.Sprint(DefaultStyles.Keyword, workspaceLabel+" (focused)")
			}

			output += fmt.Sprintf("%s:\n%s\n\n", workspaceLabel, m.inputs[i].View())
		}
	}

	output += "\nPress TAB to cycle between text boxes. Type your responses and press ENTER to send.\n"
	output += "Press ESC or q to quit."

	return output
}
