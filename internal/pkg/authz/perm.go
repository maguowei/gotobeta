package authz

// 权限编码目录（动作授权）：跨模块共享词表的唯一真值源。
// workspace 平台权限 seed、RBAC 策略与各消费模块统一引用此处，避免字面量漂移。
const (
	PermWorkspaceManage  = "workspace.manage"
	PermMemberInvite     = "member.invite"
	PermMemberRemove     = "member.remove"
	PermRoleManage       = "role.manage"
	PermChannelCreate    = "channel.create"
	PermChannelArchive   = "channel.archive"
	PermConversationRead = "conversation.read"
	PermMessageSend      = "message.send"
	PermMessageRecall    = "message.recall"
	PermMessageReact     = "message.react"
	PermBotManage        = "bot.manage"
)
