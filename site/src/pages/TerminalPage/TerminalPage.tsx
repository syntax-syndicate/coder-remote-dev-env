import "@xterm/xterm/css/xterm.css";
import type { Interpolation, Theme } from "@emotion/react";
import { CanvasAddon } from "@xterm/addon-canvas";
import { FitAddon } from "@xterm/addon-fit";
import { Unicode11Addon } from "@xterm/addon-unicode11";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { WebglAddon } from "@xterm/addon-webgl";
import { Terminal } from "@xterm/xterm";
import { deploymentConfig } from "api/queries/deployment";
import { appearanceSettings } from "api/queries/users";
import {
	workspaceByOwnerAndName,
	workspaceUsage,
} from "api/queries/workspaces";
import { useProxy } from "contexts/ProxyContext";
import { ThemeOverride } from "contexts/ThemeProvider";
import { useEmbeddedMetadata } from "hooks/useEmbeddedMetadata";
import { type FC, useCallback, useEffect, useRef, useState } from "react";
import { Helmet } from "react-helmet-async";
import { useQuery } from "react-query";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import themes from "theme";
import { DEFAULT_TERMINAL_FONT, terminalFonts } from "theme/constants";
import { pageTitle } from "utils/page";
import { openMaybePortForwardedURL } from "utils/portForward";
import { terminalWebsocketUrl } from "utils/terminal";
import { getMatchingAgentOrFirst } from "utils/workspace";
import { v4 as uuidv4 } from "uuid";
import { TerminalAlerts } from "./TerminalAlerts";
import type { ConnectionStatus } from "./types";

export const Language = {
	workspaceErrorMessagePrefix: "Unable to fetch workspace: ",
	workspaceAgentErrorMessagePrefix: "Unable to fetch workspace agent: ",
	websocketErrorMessagePrefix: "WebSocket failed: ",
};

const TerminalPage: FC = () => {
	// Maybe one day we'll support a light themed terminal, but terminal coloring
	// is notably a pain because of assumptions certain programs might make about your
	// background color.
	const theme = themes.dark;
	const navigate = useNavigate();
	const { proxy, proxyLatencies } = useProxy();
	const params = useParams() as { username: string; workspace: string };
	const username = params.username.replace("@", "");
	const terminalWrapperRef = useRef<HTMLDivElement>(null);
	// The terminal is maintained as a state to trigger certain effects when it
	// updates.
	const [terminal, setTerminal] = useState<Terminal>();
	const [connectionStatus, setConnectionStatus] =
		useState<ConnectionStatus>("initializing");
	const [searchParams] = useSearchParams();
	const isDebugging = searchParams.has("debug");
	// The reconnection token is a unique token that identifies
	// a terminal session. It's generated by the client to reduce
	// a round-trip, and must be a UUIDv4.
	const reconnectionToken = searchParams.get("reconnect") ?? uuidv4();
	const command = searchParams.get("command") || undefined;
	const containerName = searchParams.get("container") || undefined;
	const containerUser = searchParams.get("container_user") || undefined;
	// The workspace name is in the format:
	// <workspace name>[.<agent name>]
	const workspaceNameParts = params.workspace?.split(".");
	const workspace = useQuery(
		workspaceByOwnerAndName(username, workspaceNameParts?.[0]),
	);
	const workspaceAgent = workspace.data
		? getMatchingAgentOrFirst(workspace.data, workspaceNameParts?.[1])
		: undefined;
	const selectedProxy = proxy.proxy;
	const latency = selectedProxy ? proxyLatencies[selectedProxy.id] : undefined;

	const config = useQuery(deploymentConfig());
	const renderer = config.data?.config.web_terminal_renderer;

	// Periodically report workspace usage.
	useQuery(
		workspaceUsage({
			usageApp: "reconnecting-pty",
			connectionStatus,
			workspaceId: workspace.data?.id,
			agentId: workspaceAgent?.id,
		}),
	);

	// handleWebLink handles opening of URLs in the terminal!
	const handleWebLink = useCallback(
		(uri: string) => {
			openMaybePortForwardedURL(
				uri,
				proxy.preferredWildcardHostname,
				workspaceAgent?.name,
				workspace.data?.name,
				username,
			);
		},
		[workspaceAgent, workspace.data, username, proxy.preferredWildcardHostname],
	);
	const handleWebLinkRef = useRef(handleWebLink);
	useEffect(() => {
		handleWebLinkRef.current = handleWebLink;
	}, [handleWebLink]);

	const { metadata } = useEmbeddedMetadata();
	const appearanceSettingsQuery = useQuery(
		appearanceSettings(metadata.userAppearance),
	);
	const currentTerminalFont =
		appearanceSettingsQuery.data?.terminal_font || DEFAULT_TERMINAL_FONT;

	// Create the terminal!
	const fitAddonRef = useRef<FitAddon>();
	useEffect(() => {
		if (!terminalWrapperRef.current || config.isLoading) {
			return;
		}
		const terminal = new Terminal({
			allowProposedApi: true,
			allowTransparency: true,
			disableStdin: false,
			fontFamily: terminalFonts[currentTerminalFont],
			fontSize: 16,
			theme: {
				background: theme.palette.background.default,
			},
		});
		if (renderer === "webgl") {
			terminal.loadAddon(new WebglAddon());
		} else if (renderer === "canvas") {
			terminal.loadAddon(new CanvasAddon());
		}
		const fitAddon = new FitAddon();
		fitAddonRef.current = fitAddon;
		terminal.loadAddon(fitAddon);
		terminal.loadAddon(new Unicode11Addon());
		terminal.unicode.activeVersion = "11";
		terminal.loadAddon(
			new WebLinksAddon((_, uri) => {
				handleWebLinkRef.current(uri);
			}),
		);

		terminal.open(terminalWrapperRef.current);

		// We have to fit twice here. It's unknown why, but the first fit will
		// overflow slightly in some scenarios. Applying a second fit resolves this.
		fitAddon.fit();
		fitAddon.fit();

		// This will trigger a resize event on the terminal.
		const listener = () => fitAddon.fit();
		window.addEventListener("resize", listener);

		// Terminal is correctly sized and is ready to be used.
		setTerminal(terminal);

		return () => {
			window.removeEventListener("resize", listener);
			terminal.dispose();
		};
	}, [
		config.isLoading,
		renderer,
		theme.palette.background.default,
		currentTerminalFont,
	]);

	// Updates the reconnection token into the URL if necessary.
	useEffect(() => {
		if (searchParams.get("reconnect") === reconnectionToken) {
			return;
		}
		searchParams.set("reconnect", reconnectionToken);
		navigate(
			{
				search: searchParams.toString(),
			},
			{
				replace: true,
			},
		);
	}, [navigate, reconnectionToken, searchParams]);

	// Hook up the terminal through a web socket.
	useEffect(() => {
		if (!terminal) {
			return;
		}

		// The terminal should be cleared on each reconnect
		// because all data is re-rendered from the backend.
		terminal.clear();

		// Focusing on connection allows users to reload the page and start
		// typing immediately.
		terminal.focus();

		// Disable input while we connect.
		terminal.options.disableStdin = true;

		// Show a message if we failed to find the workspace or agent.
		if (workspace.isLoading) {
			return;
		}

		if (workspace.error instanceof Error) {
			terminal.writeln(
				Language.workspaceErrorMessagePrefix + workspace.error.message,
			);
			setConnectionStatus("disconnected");
			return;
		}

		if (!workspaceAgent) {
			terminal.writeln(
				`${Language.workspaceAgentErrorMessagePrefix}no agent found with ID, is the workspace started?`,
			);
			setConnectionStatus("disconnected");
			return;
		}

		// Hook up terminal events to the websocket.
		let websocket: WebSocket | null;
		const disposers = [
			terminal.onData((data) => {
				websocket?.send(
					new TextEncoder().encode(JSON.stringify({ data: data })),
				);
			}),
			terminal.onResize((event) => {
				websocket?.send(
					new TextEncoder().encode(
						JSON.stringify({
							height: event.rows,
							width: event.cols,
						}),
					),
				);
			}),
		];

		let disposed = false;

		// Open the web socket and hook it up to the terminal.
		terminalWebsocketUrl(
			proxy.preferredPathAppURL,
			reconnectionToken,
			workspaceAgent.id,
			command,
			terminal.rows,
			terminal.cols,
			containerName,
			containerUser,
		)
			.then((url) => {
				if (disposed) {
					return; // Unmounted while we waited for the async call.
				}
				websocket = new WebSocket(url);
				websocket.binaryType = "arraybuffer";
				websocket.addEventListener("open", () => {
					// Now that we are connected, allow user input.
					terminal.options = {
						disableStdin: false,
						windowsMode: workspaceAgent?.operating_system === "windows",
					};
					// Send the initial size.
					websocket?.send(
						new TextEncoder().encode(
							JSON.stringify({
								height: terminal.rows,
								width: terminal.cols,
							}),
						),
					);
					setConnectionStatus("connected");
				});
				websocket.addEventListener("error", () => {
					terminal.options.disableStdin = true;
					terminal.writeln(
						`${Language.websocketErrorMessagePrefix}socket errored`,
					);
					setConnectionStatus("disconnected");
				});
				websocket.addEventListener("close", () => {
					terminal.options.disableStdin = true;
					setConnectionStatus("disconnected");
				});
				websocket.addEventListener("message", (event) => {
					if (typeof event.data === "string") {
						// This exclusively occurs when testing.
						// "jest-websocket-mock" doesn't support ArrayBuffer.
						terminal.write(event.data);
					} else {
						terminal.write(new Uint8Array(event.data));
					}
				});
			})
			.catch((error) => {
				if (disposed) {
					return; // Unmounted while we waited for the async call.
				}
				terminal.writeln(Language.websocketErrorMessagePrefix + error.message);
				setConnectionStatus("disconnected");
			});

		return () => {
			disposed = true; // Could use AbortController instead?
			for (const d of disposers) {
				d.dispose();
			}
			websocket?.close(1000);
		};
	}, [
		command,
		proxy.preferredPathAppURL,
		reconnectionToken,
		terminal,
		workspace.error,
		workspace.isLoading,
		workspaceAgent,
		containerName,
		containerUser,
	]);

	return (
		<ThemeOverride theme={theme}>
			<Helmet>
				<title>
					{workspace.data
						? pageTitle(
								`Terminal · ${workspace.data.owner_name}/${workspace.data.name}`,
							)
						: ""}
				</title>
			</Helmet>
			<div
				css={{ display: "flex", flexDirection: "column", height: "100vh" }}
				data-status={connectionStatus}
			>
				<TerminalAlerts
					agent={workspaceAgent}
					status={connectionStatus}
					onAlertChange={() => {
						fitAddonRef.current?.fit();
					}}
				/>
				<div
					css={styles.terminal}
					ref={terminalWrapperRef}
					data-testid="terminal"
				/>
			</div>

			{latency && isDebugging && (
				<span
					css={{
						position: "absolute",
						bottom: 24,
						right: 24,
						color: theme.palette.text.disabled,
						fontSize: 14,
					}}
				>
					Latency: {latency.latencyMS.toFixed(0)}ms
				</span>
			)}
		</ThemeOverride>
	);
};

const styles = {
	terminal: (theme) => ({
		width: "100%",
		overflow: "hidden",
		backgroundColor: theme.palette.background.paper,
		flex: 1,
		// These styles attempt to mimic the VS Code scrollbar.
		"& .xterm": {
			padding: 4,
			width: "100%",
			height: "100%",
		},
		"& .xterm-viewport": {
			// This is required to force full-width on the terminal.
			// Otherwise there's a small white bar to the right of the scrollbar.
			width: "auto !important",
		},
		"& .xterm-viewport::-webkit-scrollbar": {
			width: "10px",
		},
		"& .xterm-viewport::-webkit-scrollbar-track": {
			backgroundColor: "inherit",
		},
		"& .xterm-viewport::-webkit-scrollbar-thumb": {
			minHeight: 20,
			backgroundColor: "rgba(255, 255, 255, 0.18)",
		},
	}),
} satisfies Record<string, Interpolation<Theme>>;

export default TerminalPage;
