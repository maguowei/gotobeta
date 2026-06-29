package imrt

// Connection 是一条可写帧的实时连接（由 WS adapter 实现）。
//
// 放在共享内核以便 infra/hub（注册表）与 adapter/ws（连接）共用同一契约，
// 而互不直接依赖（符合分层边界）。
type Connection interface {
	// Send 非阻塞投递一帧；写队列已满或已关闭时丢弃（推送尽力而为）。
	Send(frame []byte)
}

// Registry 是连接注册表（由 infra/hub 实现），供 WS 网关注册/注销连接。
type Registry interface {
	// Register 注册一条连接；超过连接上限时拒绝并返回 false。
	Register(userID int64, c Connection) bool
	Unregister(userID int64, c Connection)
}
