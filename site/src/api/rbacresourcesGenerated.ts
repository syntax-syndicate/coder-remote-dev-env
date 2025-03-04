// Code generated by typegen/main.go. DO NOT EDIT.

import type { RBACAction, RBACResource } from "./typesGenerated";

// RBACResourceActions maps RBAC resources to their possible actions.
// Descriptions are included to document the purpose of each action.
// Source is in 'coderd/rbac/policy/policy.go'.
export const RBACResourceActions: Partial<
	Record<RBACResource, Partial<Record<RBACAction, string>>>
> = {
	api_key: {
		create: "create an api key",
		delete: "delete an api key",
		read: "read api key details (secrets are not stored)",
		update: "update an api key, eg expires",
	},
	assign_org_role: {
		assign: "assign org scoped roles",
		create: "create/delete custom roles within an organization",
		delete: "delete roles within an organization",
		read: "view what roles are assignable within an organization",
		unassign: "unassign org scoped roles",
		update: "edit custom roles within an organization",
	},
	assign_role: {
		assign: "assign user roles",
		read: "view what roles are assignable",
		unassign: "unassign user roles",
	},
	audit_log: {
		create: "create new audit log entries",
		read: "read audit logs",
	},
	crypto_key: {
		create: "create crypto keys",
		delete: "delete crypto keys",
		read: "read crypto keys",
		update: "update crypto keys",
	},
	debug_info: {
		read: "access to debug routes",
	},
	deployment_config: {
		read: "read deployment config",
		update: "updating health information",
	},
	deployment_stats: {
		read: "read deployment stats",
	},
	file: {
		create: "create a file",
		read: "read files",
	},
	group: {
		create: "create a group",
		delete: "delete a group",
		read: "read groups",
		update: "update a group",
	},
	group_member: {
		read: "read group members",
	},
	idpsync_settings: {
		read: "read IdP sync settings",
		update: "update IdP sync settings",
	},
	inbox_notification: {
		create: "create inbox notifications",
		read: "read inbox notifications",
		update: "update inbox notifications",
	},
	license: {
		create: "create a license",
		delete: "delete license",
		read: "read licenses",
	},
	notification_message: {
		create: "create notification messages",
		delete: "delete notification messages",
		read: "read notification messages",
		update: "update notification messages",
	},
	notification_preference: {
		read: "read notification preferences",
		update: "update notification preferences",
	},
	notification_template: {
		read: "read notification templates",
		update: "update notification templates",
	},
	oauth2_app: {
		create: "make an OAuth2 app",
		delete: "delete an OAuth2 app",
		read: "read OAuth2 apps",
		update: "update the properties of the OAuth2 app",
	},
	oauth2_app_code_token: {
		create: "create an OAuth2 app code token",
		delete: "delete an OAuth2 app code token",
		read: "read an OAuth2 app code token",
	},
	oauth2_app_secret: {
		create: "create an OAuth2 app secret",
		delete: "delete an OAuth2 app secret",
		read: "read an OAuth2 app secret",
		update: "update an OAuth2 app secret",
	},
	organization: {
		create: "create an organization",
		delete: "delete an organization",
		read: "read organizations",
		update: "update an organization",
	},
	organization_member: {
		create: "create an organization member",
		delete: "delete member",
		read: "read member",
		update: "update an organization member",
	},
	provisioner_daemon: {
		create: "create a provisioner daemon/key",
		delete: "delete a provisioner daemon/key",
		read: "read provisioner daemon",
		update: "update a provisioner daemon",
	},
	provisioner_jobs: {
		read: "read provisioner jobs",
	},
	replicas: {
		read: "read replicas",
	},
	system: {
		create: "create system resources",
		delete: "delete system resources",
		read: "view system resources",
		update: "update system resources",
	},
	tailnet_coordinator: {
		create: "create a Tailnet coordinator",
		delete: "delete a Tailnet coordinator",
		read: "view info about a Tailnet coordinator",
		update: "update a Tailnet coordinator",
	},
	template: {
		create: "create a template",
		delete: "delete a template",
		read: "read template",
		update: "update a template",
		use: "use the template to initially create a workspace, then workspace lifecycle permissions take over",
		view_insights: "view insights",
	},
	user: {
		create: "create a new user",
		delete: "delete an existing user",
		read: "read user data",
		read_personal: "read personal user data like user settings and auth links",
		update: "update an existing user",
		update_personal: "update personal data",
	},
	workspace: {
		application_connect: "connect to workspace apps via browser",
		create: "create a new workspace",
		delete: "delete workspace",
		read: "read workspace data to view on the UI",
		ssh: "ssh into a given workspace",
		start: "allows starting a workspace",
		stop: "allows stopping a workspace",
		update: "edit workspace settings (scheduling, permissions, parameters)",
	},
	workspace_agent_resource_monitor: {
		create: "create workspace agent resource monitor",
		read: "read workspace agent resource monitor",
		update: "update workspace agent resource monitor",
	},
	workspace_dormant: {
		application_connect: "connect to workspace apps via browser",
		create: "create a new workspace",
		delete: "delete workspace",
		read: "read workspace data to view on the UI",
		ssh: "ssh into a given workspace",
		start: "allows starting a workspace",
		stop: "allows stopping a workspace",
		update: "edit workspace settings (scheduling, permissions, parameters)",
	},
	workspace_proxy: {
		create: "create a workspace proxy",
		delete: "delete a workspace proxy",
		read: "read and use a workspace proxy",
		update: "update a workspace proxy",
	},
};
