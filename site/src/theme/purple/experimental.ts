import type { NewTheme } from "../experimental";
import tw from "../tailwindColors";
import darkExperimental from "../dark/experimental";

// Create purple experimental theme by extending dark theme
const experimental: NewTheme = {
	// Start with dark theme
	...darkExperimental,
	
	// Override with purple-specific styles
	l1: {
		background: tw.purple[950],
		outline: tw.purple[700],
		text: tw.white,
		fill: {
			solid: tw.purple[600],
			outline: tw.purple[600],
			text: tw.white,
		},
	},

	l2: {
		background: tw.purple[900],
		outline: tw.purple[700],
		text: tw.zinc[50],
		fill: {
			solid: tw.purple[500],
			outline: tw.purple[500],
			text: tw.white,
		},
		disabled: {
			background: tw.purple[900],
			outline: tw.purple[700],
			text: tw.zinc[200],
			fill: {
				solid: tw.purple[500],
				outline: tw.purple[500],
				text: tw.white,
			},
		},
		hover: {
			background: tw.purple[600],
			outline: tw.purple[500],
			text: tw.white,
			fill: {
				solid: tw.purple[400],
				outline: tw.purple[400],
				text: tw.white,
			},
		},
	},

	pillDefault: {
		background: tw.purple[800],
		outline: tw.purple[700],
		text: tw.white,
	},
	
	avatar: {
		background: tw.purple[200],
		text: tw.purple[900],
		border: tw.purple[300],
	},
};

export default experimental;