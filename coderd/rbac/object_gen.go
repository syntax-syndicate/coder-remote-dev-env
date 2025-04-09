// Code generated by typegen/main.go. DO NOT EDIT.
package rbac

import "github.com/coder/coder/v2/coderd/rbac/policy"

// Objecter returns the RBAC object for itself.
type Objecter interface {
	RBACObject() Object
}

var (
	// ResourceWildcard
	// Valid Actions
	ResourceWildcard = Object{
		Type: "*",
	}

	// ResourceApiKey
	// Valid Actions
	//  - "ActionCreate" :: create an api key
	//  - "ActionDelete" :: delete an api key
	//  - "ActionRead" :: read api key details (secrets are not stored)
	//  - "ActionUpdate" :: update an api key, eg expires
	ResourceApiKey = Object{
		Type: "api_key",
	}

	// ResourceAssignOrgRole
	// Valid Actions
	//  - "ActionAssign" :: assign org scoped roles
	//  - "ActionCreate" :: create/delete custom roles within an organization
	//  - "ActionDelete" :: delete roles within an organization
	//  - "ActionRead" :: view what roles are assignable within an organization
	//  - "ActionUnassign" :: unassign org scoped roles
	//  - "ActionUpdate" :: edit custom roles within an organization
	ResourceAssignOrgRole = Object{
		Type: "assign_org_role",
	}

	// ResourceAssignRole
	// Valid Actions
	//  - "ActionAssign" :: assign user roles
	//  - "ActionRead" :: view what roles are assignable
	//  - "ActionUnassign" :: unassign user roles
	ResourceAssignRole = Object{
		Type: "assign_role",
	}

	// ResourceAuditLog
	// Valid Actions
	//  - "ActionCreate" :: create new audit log entries
	//  - "ActionRead" :: read audit logs
	ResourceAuditLog = Object{
		Type: "audit_log",
	}

	// ResourceChat
	// Valid Actions
	//  - "ActionCreate" :: create a chat
	//  - "ActionDelete" :: delete a chat
	//  - "ActionRead" :: read a chat
	//  - "ActionUpdate" :: update a chat
	ResourceChat = Object{
		Type: "chat",
	}

	// ResourceCryptoKey
	// Valid Actions
	//  - "ActionCreate" :: create crypto keys
	//  - "ActionDelete" :: delete crypto keys
	//  - "ActionRead" :: read crypto keys
	//  - "ActionUpdate" :: update crypto keys
	ResourceCryptoKey = Object{
		Type: "crypto_key",
	}

	// ResourceDebugInfo
	// Valid Actions
	//  - "ActionRead" :: access to debug routes
	ResourceDebugInfo = Object{
		Type: "debug_info",
	}

	// ResourceDeploymentConfig
	// Valid Actions
	//  - "ActionRead" :: read deployment config
	//  - "ActionUpdate" :: updating health information
	ResourceDeploymentConfig = Object{
		Type: "deployment_config",
	}

	// ResourceDeploymentStats
	// Valid Actions
	//  - "ActionRead" :: read deployment stats
	ResourceDeploymentStats = Object{
		Type: "deployment_stats",
	}

	// ResourceFile
	// Valid Actions
	//  - "ActionCreate" :: create a file
	//  - "ActionRead" :: read files
	ResourceFile = Object{
		Type: "file",
	}

	// ResourceGroup
	// Valid Actions
	//  - "ActionCreate" :: create a group
	//  - "ActionDelete" :: delete a group
	//  - "ActionRead" :: read groups
	//  - "ActionUpdate" :: update a group
	ResourceGroup = Object{
		Type: "group",
	}

	// ResourceGroupMember
	// Valid Actions
	//  - "ActionRead" :: read group members
	ResourceGroupMember = Object{
		Type: "group_member",
	}

	// ResourceIdpsyncSettings
	// Valid Actions
	//  - "ActionRead" :: read IdP sync settings
	//  - "ActionUpdate" :: update IdP sync settings
	ResourceIdpsyncSettings = Object{
		Type: "idpsync_settings",
	}

	// ResourceInboxNotification
	// Valid Actions
	//  - "ActionCreate" :: create inbox notifications
	//  - "ActionRead" :: read inbox notifications
	//  - "ActionUpdate" :: update inbox notifications
	ResourceInboxNotification = Object{
		Type: "inbox_notification",
	}

	// ResourceLicense
	// Valid Actions
	//  - "ActionCreate" :: create a license
	//  - "ActionDelete" :: delete license
	//  - "ActionRead" :: read licenses
	ResourceLicense = Object{
		Type: "license",
	}

	// ResourceNotificationMessage
	// Valid Actions
	//  - "ActionCreate" :: create notification messages
	//  - "ActionDelete" :: delete notification messages
	//  - "ActionRead" :: read notification messages
	//  - "ActionUpdate" :: update notification messages
	ResourceNotificationMessage = Object{
		Type: "notification_message",
	}

	// ResourceNotificationPreference
	// Valid Actions
	//  - "ActionRead" :: read notification preferences
	//  - "ActionUpdate" :: update notification preferences
	ResourceNotificationPreference = Object{
		Type: "notification_preference",
	}

	// ResourceNotificationTemplate
	// Valid Actions
	//  - "ActionRead" :: read notification templates
	//  - "ActionUpdate" :: update notification templates
	ResourceNotificationTemplate = Object{
		Type: "notification_template",
	}

	// ResourceOauth2App
	// Valid Actions
	//  - "ActionCreate" :: make an OAuth2 app
	//  - "ActionDelete" :: delete an OAuth2 app
	//  - "ActionRead" :: read OAuth2 apps
	//  - "ActionUpdate" :: update the properties of the OAuth2 app
	ResourceOauth2App = Object{
		Type: "oauth2_app",
	}

	// ResourceOauth2AppCodeToken
	// Valid Actions
	//  - "ActionCreate" :: create an OAuth2 app code token
	//  - "ActionDelete" :: delete an OAuth2 app code token
	//  - "ActionRead" :: read an OAuth2 app code token
	ResourceOauth2AppCodeToken = Object{
		Type: "oauth2_app_code_token",
	}

	// ResourceOauth2AppSecret
	// Valid Actions
	//  - "ActionCreate" :: create an OAuth2 app secret
	//  - "ActionDelete" :: delete an OAuth2 app secret
	//  - "ActionRead" :: read an OAuth2 app secret
	//  - "ActionUpdate" :: update an OAuth2 app secret
	ResourceOauth2AppSecret = Object{
		Type: "oauth2_app_secret",
	}

	// ResourceOrganization
	// Valid Actions
	//  - "ActionCreate" :: create an organization
	//  - "ActionDelete" :: delete an organization
	//  - "ActionRead" :: read organizations
	//  - "ActionUpdate" :: update an organization
	ResourceOrganization = Object{
		Type: "organization",
	}

	// ResourceOrganizationMember
	// Valid Actions
	//  - "ActionCreate" :: create an organization member
	//  - "ActionDelete" :: delete member
	//  - "ActionRead" :: read member
	//  - "ActionUpdate" :: update an organization member
	ResourceOrganizationMember = Object{
		Type: "organization_member",
	}

	// ResourceProvisionerDaemon
	// Valid Actions
	//  - "ActionCreate" :: create a provisioner daemon/key
	//  - "ActionDelete" :: delete a provisioner daemon/key
	//  - "ActionRead" :: read provisioner daemon
	//  - "ActionUpdate" :: update a provisioner daemon
	ResourceProvisionerDaemon = Object{
		Type: "provisioner_daemon",
	}

	// ResourceProvisionerJobs
	// Valid Actions
	//  - "ActionRead" :: read provisioner jobs
	ResourceProvisionerJobs = Object{
		Type: "provisioner_jobs",
	}

	// ResourceReplicas
	// Valid Actions
	//  - "ActionRead" :: read replicas
	ResourceReplicas = Object{
		Type: "replicas",
	}

	// ResourceSystem
	// Valid Actions
	//  - "ActionCreate" :: create system resources
	//  - "ActionDelete" :: delete system resources
	//  - "ActionRead" :: view system resources
	//  - "ActionUpdate" :: update system resources
	// DEPRECATED: New resources should be created for new things, rather than adding them to System, which has become
	//             an unmanaged collection of things that don't relate to one another. We can't effectively enforce
	//             least privilege access control when unrelated resources are grouped together.
	ResourceSystem = Object{
		Type: "system",
	}

	// ResourceTailnetCoordinator
	// Valid Actions
	//  - "ActionCreate" :: create a Tailnet coordinator
	//  - "ActionDelete" :: delete a Tailnet coordinator
	//  - "ActionRead" :: view info about a Tailnet coordinator
	//  - "ActionUpdate" :: update a Tailnet coordinator
	ResourceTailnetCoordinator = Object{
		Type: "tailnet_coordinator",
	}

	// ResourceTemplate
	// Valid Actions
	//  - "ActionCreate" :: create a template
	//  - "ActionDelete" :: delete a template
	//  - "ActionRead" :: read template
	//  - "ActionUpdate" :: update a template
	//  - "ActionUse" :: use the template to initially create a workspace, then workspace lifecycle permissions take over
	//  - "ActionViewInsights" :: view insights
	ResourceTemplate = Object{
		Type: "template",
	}

	// ResourceUser
	// Valid Actions
	//  - "ActionCreate" :: create a new user
	//  - "ActionDelete" :: delete an existing user
	//  - "ActionRead" :: read user data
	//  - "ActionReadPersonal" :: read personal user data like user settings and auth links
	//  - "ActionUpdate" :: update an existing user
	//  - "ActionUpdatePersonal" :: update personal data
	ResourceUser = Object{
		Type: "user",
	}

	// ResourceWebpushSubscription
	// Valid Actions
	//  - "ActionCreate" :: create webpush subscriptions
	//  - "ActionDelete" :: delete webpush subscriptions
	//  - "ActionRead" :: read webpush subscriptions
	ResourceWebpushSubscription = Object{
		Type: "webpush_subscription",
	}

	// ResourceWorkspace
	// Valid Actions
	//  - "ActionApplicationConnect" :: connect to workspace apps via browser
	//  - "ActionCreate" :: create a new workspace
	//  - "ActionDelete" :: delete workspace
	//  - "ActionRead" :: read workspace data to view on the UI
	//  - "ActionSSH" :: ssh into a given workspace
	//  - "ActionWorkspaceStart" :: allows starting a workspace
	//  - "ActionWorkspaceStop" :: allows stopping a workspace
	//  - "ActionUpdate" :: edit workspace settings (scheduling, permissions, parameters)
	ResourceWorkspace = Object{
		Type: "workspace",
	}

	// ResourceWorkspaceAgentDevcontainers
	// Valid Actions
	//  - "ActionCreate" :: create workspace agent devcontainers
	ResourceWorkspaceAgentDevcontainers = Object{
		Type: "workspace_agent_devcontainers",
	}

	// ResourceWorkspaceAgentResourceMonitor
	// Valid Actions
	//  - "ActionCreate" :: create workspace agent resource monitor
	//  - "ActionRead" :: read workspace agent resource monitor
	//  - "ActionUpdate" :: update workspace agent resource monitor
	ResourceWorkspaceAgentResourceMonitor = Object{
		Type: "workspace_agent_resource_monitor",
	}

	// ResourceWorkspaceDormant
	// Valid Actions
	//  - "ActionApplicationConnect" :: connect to workspace apps via browser
	//  - "ActionCreate" :: create a new workspace
	//  - "ActionDelete" :: delete workspace
	//  - "ActionRead" :: read workspace data to view on the UI
	//  - "ActionSSH" :: ssh into a given workspace
	//  - "ActionWorkspaceStart" :: allows starting a workspace
	//  - "ActionWorkspaceStop" :: allows stopping a workspace
	//  - "ActionUpdate" :: edit workspace settings (scheduling, permissions, parameters)
	ResourceWorkspaceDormant = Object{
		Type: "workspace_dormant",
	}

	// ResourceWorkspaceProxy
	// Valid Actions
	//  - "ActionCreate" :: create a workspace proxy
	//  - "ActionDelete" :: delete a workspace proxy
	//  - "ActionRead" :: read and use a workspace proxy
	//  - "ActionUpdate" :: update a workspace proxy
	ResourceWorkspaceProxy = Object{
		Type: "workspace_proxy",
	}
)

func AllResources() []Objecter {
	return []Objecter{
		ResourceWildcard,
		ResourceApiKey,
		ResourceAssignOrgRole,
		ResourceAssignRole,
		ResourceAuditLog,
		ResourceChat,
		ResourceCryptoKey,
		ResourceDebugInfo,
		ResourceDeploymentConfig,
		ResourceDeploymentStats,
		ResourceFile,
		ResourceGroup,
		ResourceGroupMember,
		ResourceIdpsyncSettings,
		ResourceInboxNotification,
		ResourceLicense,
		ResourceNotificationMessage,
		ResourceNotificationPreference,
		ResourceNotificationTemplate,
		ResourceOauth2App,
		ResourceOauth2AppCodeToken,
		ResourceOauth2AppSecret,
		ResourceOrganization,
		ResourceOrganizationMember,
		ResourceProvisionerDaemon,
		ResourceProvisionerJobs,
		ResourceReplicas,
		ResourceSystem,
		ResourceTailnetCoordinator,
		ResourceTemplate,
		ResourceUser,
		ResourceWebpushSubscription,
		ResourceWorkspace,
		ResourceWorkspaceAgentDevcontainers,
		ResourceWorkspaceAgentResourceMonitor,
		ResourceWorkspaceDormant,
		ResourceWorkspaceProxy,
	}
}

func AllActions() []policy.Action {
	return []policy.Action{
		policy.ActionApplicationConnect,
		policy.ActionAssign,
		policy.ActionCreate,
		policy.ActionDelete,
		policy.ActionRead,
		policy.ActionReadPersonal,
		policy.ActionSSH,
		policy.ActionUnassign,
		policy.ActionUpdate,
		policy.ActionUpdatePersonal,
		policy.ActionUse,
		policy.ActionViewInsights,
		policy.ActionWorkspaceStart,
		policy.ActionWorkspaceStop,
	}
}
