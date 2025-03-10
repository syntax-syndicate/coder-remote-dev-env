// Code generated by typegen/main.go. DO NOT EDIT.
package codersdk

type RBACResource string

const (
	ResourceWildcard                      RBACResource = "*"
	ResourceApiKey                        RBACResource = "api_key"
	ResourceAssignOrgRole                 RBACResource = "assign_org_role"
	ResourceAssignRole                    RBACResource = "assign_role"
	ResourceAuditLog                      RBACResource = "audit_log"
	ResourceCryptoKey                     RBACResource = "crypto_key"
	ResourceDebugInfo                     RBACResource = "debug_info"
	ResourceDeploymentConfig              RBACResource = "deployment_config"
	ResourceDeploymentStats               RBACResource = "deployment_stats"
	ResourceFile                          RBACResource = "file"
	ResourceGroup                         RBACResource = "group"
	ResourceGroupMember                   RBACResource = "group_member"
	ResourceIdpsyncSettings               RBACResource = "idpsync_settings"
	ResourceInboxNotification             RBACResource = "inbox_notification"
	ResourceLicense                       RBACResource = "license"
	ResourceNotificationMessage           RBACResource = "notification_message"
	ResourceNotificationPreference        RBACResource = "notification_preference"
	ResourceNotificationTemplate          RBACResource = "notification_template"
	ResourceOauth2App                     RBACResource = "oauth2_app"
	ResourceOauth2AppCodeToken            RBACResource = "oauth2_app_code_token"
	ResourceOauth2AppSecret               RBACResource = "oauth2_app_secret"
	ResourceOrganization                  RBACResource = "organization"
	ResourceOrganizationMember            RBACResource = "organization_member"
	ResourceProvisionerDaemon             RBACResource = "provisioner_daemon"
	ResourceProvisionerJobs               RBACResource = "provisioner_jobs"
	ResourceReplicas                      RBACResource = "replicas"
	ResourceSystem                        RBACResource = "system"
	ResourceTailnetCoordinator            RBACResource = "tailnet_coordinator"
	ResourceTemplate                      RBACResource = "template"
	ResourceUser                          RBACResource = "user"
	ResourceWorkspace                     RBACResource = "workspace"
	ResourceWorkspaceAgentResourceMonitor RBACResource = "workspace_agent_resource_monitor"
	ResourceWorkspaceDormant              RBACResource = "workspace_dormant"
	ResourceWorkspaceProxy                RBACResource = "workspace_proxy"
)

type RBACAction string

const (
	ActionApplicationConnect RBACAction = "application_connect"
	ActionAssign             RBACAction = "assign"
	ActionCreate             RBACAction = "create"
	ActionDelete             RBACAction = "delete"
	ActionRead               RBACAction = "read"
	ActionReadPersonal       RBACAction = "read_personal"
	ActionSSH                RBACAction = "ssh"
	ActionUnassign           RBACAction = "unassign"
	ActionUpdate             RBACAction = "update"
	ActionUpdatePersonal     RBACAction = "update_personal"
	ActionUse                RBACAction = "use"
	ActionViewInsights       RBACAction = "view_insights"
	ActionWorkspaceStart     RBACAction = "start"
	ActionWorkspaceStop      RBACAction = "stop"
)

// RBACResourceActions is the mapping of resources to which actions are valid for
// said resource type.
var RBACResourceActions = map[RBACResource][]RBACAction{
	ResourceWildcard:                      {},
	ResourceApiKey:                        {ActionCreate, ActionDelete, ActionRead, ActionUpdate},
	ResourceAssignOrgRole:                 {ActionAssign, ActionCreate, ActionDelete, ActionRead, ActionUnassign, ActionUpdate},
	ResourceAssignRole:                    {ActionAssign, ActionRead, ActionUnassign},
	ResourceAuditLog:                      {ActionCreate, ActionRead},
	ResourceCryptoKey:                     {ActionCreate, ActionDelete, ActionRead, ActionUpdate},
	ResourceDebugInfo:                     {ActionRead},
	ResourceDeploymentConfig:              {ActionRead, ActionUpdate},
	ResourceDeploymentStats:               {ActionRead},
	ResourceFile:                          {ActionCreate, ActionRead},
	ResourceGroup:                         {ActionCreate, ActionDelete, ActionRead, ActionUpdate},
	ResourceGroupMember:                   {ActionRead},
	ResourceIdpsyncSettings:               {ActionRead, ActionUpdate},
	ResourceInboxNotification:             {ActionCreate, ActionRead, ActionUpdate},
	ResourceLicense:                       {ActionCreate, ActionDelete, ActionRead},
	ResourceNotificationMessage:           {ActionCreate, ActionDelete, ActionRead, ActionUpdate},
	ResourceNotificationPreference:        {ActionRead, ActionUpdate},
	ResourceNotificationTemplate:          {ActionRead, ActionUpdate},
	ResourceOauth2App:                     {ActionCreate, ActionDelete, ActionRead, ActionUpdate},
	ResourceOauth2AppCodeToken:            {ActionCreate, ActionDelete, ActionRead},
	ResourceOauth2AppSecret:               {ActionCreate, ActionDelete, ActionRead, ActionUpdate},
	ResourceOrganization:                  {ActionCreate, ActionDelete, ActionRead, ActionUpdate},
	ResourceOrganizationMember:            {ActionCreate, ActionDelete, ActionRead, ActionUpdate},
	ResourceProvisionerDaemon:             {ActionCreate, ActionDelete, ActionRead, ActionUpdate},
	ResourceProvisionerJobs:               {ActionRead},
	ResourceReplicas:                      {ActionRead},
	ResourceSystem:                        {ActionCreate, ActionDelete, ActionRead, ActionUpdate},
	ResourceTailnetCoordinator:            {ActionCreate, ActionDelete, ActionRead, ActionUpdate},
	ResourceTemplate:                      {ActionCreate, ActionDelete, ActionRead, ActionUpdate, ActionUse, ActionViewInsights},
	ResourceUser:                          {ActionCreate, ActionDelete, ActionRead, ActionReadPersonal, ActionUpdate, ActionUpdatePersonal},
	ResourceWorkspace:                     {ActionApplicationConnect, ActionCreate, ActionDelete, ActionRead, ActionSSH, ActionWorkspaceStart, ActionWorkspaceStop, ActionUpdate},
	ResourceWorkspaceAgentResourceMonitor: {ActionCreate, ActionRead, ActionUpdate},
	ResourceWorkspaceDormant:              {ActionApplicationConnect, ActionCreate, ActionDelete, ActionRead, ActionSSH, ActionWorkspaceStart, ActionWorkspaceStop, ActionUpdate},
	ResourceWorkspaceProxy:                {ActionCreate, ActionDelete, ActionRead, ActionUpdate},
}
