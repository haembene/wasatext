package api

import (
	"net/http"
)

// Handler returns an instance of httprouter.Router that handle APIs registered here
func (rt *_router) Handler() http.Handler {
	// Session
	rt.router.POST("/session", rt.wrap(rt.doLogin))

	// Users
	rt.router.GET("/users", rt.wrap(rt.findUsers))

	// User
	rt.router.PUT("/user/name", rt.wrap(rt.setMyUserName))
	rt.router.PUT("/user/photo", rt.wrap(rt.setMyPhoto))

	// Conversations
	rt.router.GET("/conversations", rt.wrap(rt.getMyConversations))
	rt.router.GET("/conversations/:id", rt.wrap(rt.getConversation))
	rt.router.POST("/conversations/:id", rt.wrap(rt.setConversation))

	// Messages
	// Messages
	rt.router.POST("/messages", rt.wrap(rt.sendMessage))
	rt.router.POST("/forward-message", rt.wrap(rt.forwardMessage))
	rt.router.DELETE("/messages/:id", rt.wrap(rt.deleteMessage))
	rt.router.POST("/messages/:id/r/reactions", rt.wrap(rt.commentMessage))
	rt.router.DELETE("/messages/:id/r/reactions", rt.wrap(rt.uncommentMessage))

	// Groups
	rt.router.POST("/groups", rt.wrap(rt.createGroup))
	rt.router.GET("/groups/:id/members", rt.wrap(rt.getGroupMembers))
	rt.router.POST("/groups/:id/members", rt.wrap(rt.addToGroup))
	rt.router.DELETE("/groups/:groupId/members/:userId", rt.wrap(rt.leaveGroup))
	rt.router.PUT("/groups/:id/name", rt.wrap(rt.setGroupName))
	rt.router.PUT("/groups/:id/photo", rt.wrap(rt.setGroupPhoto))

	// healthcheck
	rt.router.GET("/liveness", rt.liveness)

	return rt.router
}
