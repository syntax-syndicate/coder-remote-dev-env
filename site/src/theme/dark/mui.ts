// biome-ignore lint/nursery/noRestrictedImports: createTheme
import { createTheme } from "@mui/material/styles";
import { BODY_FONT_FAMILY, borderRadius } from "../constants";
import { components } from "../mui";
import tw from "../tailwindColors";

const muiTheme = createTheme({
	palette: {
		mode: "dark",
		primary: {
			main: tw.sky[500],
			contrastText: tw.white,
			light: tw.sky[400],
			dark: tw.sky[600],
		},
		secondary: {
			main: tw.zinc[500],
			contrastText: tw.zinc[200],
			dark: tw.zinc[400],
		},
		background: {
			default: tw.green[900],
			paper: tw.green[800],
		},
		text: {
			primary: tw.green[50],
			secondary: tw.green[200],
			disabled: tw.green[300],
		},
		divider: tw.green[700],
		warning: {
			light: tw.amber[500],
			main: tw.amber[800],
			dark: tw.amber[950],
		},
		success: {
			main: tw.green[500],
			dark: tw.green[600],
		},
		info: {
			light: tw.blue[400],
			main: tw.blue[600],
			dark: tw.blue[950],
			contrastText: tw.zinc[200],
		},
		error: {
			light: tw.red[400],
			main: tw.red[500],
			dark: tw.red[950],
			contrastText: tw.zinc[200],
		},
		action: {
			hover: tw.green[700],
		},
		neutral: {
			main: tw.zinc[50],
		},
		dots: tw.zinc[500],
	},
	typography: {
		fontFamily: BODY_FONT_FAMILY,

		body1: {
			fontSize: "1rem" /* 16px at default scaling */,
			lineHeight: "160%",
		},

		body2: {
			fontSize: "0.875rem" /* 14px at default scaling */,
			lineHeight: "160%",
		},
	},
	shape: {
		borderRadius,
	},
	components,
});

export default muiTheme;
