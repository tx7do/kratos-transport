package signalr

import (
	"github.com/google/uuid"
	"github.com/philippseith/signalr"
)

type HubID string

type Hub struct {
	signalr.Hub

	id     HubID
	server *Server
}

func NewHub(server *Server) *Hub {
	u1, _ := uuid.NewUUID()

	c := &Hub{
		id:     HubID(u1.String()),
		server: server,
	}

	return c
}

func (c *Hub) HubID() HubID {
	return c.id
}

func (c *Hub) OnConnected(connectionID string) {
}

func (c *Hub) OnDisconnected(connectionID string) {
	c.server.hubMgr.Remove(c)
}

func (c *Hub) AddToGroup(groupName, connectionID string) {
	c.Groups().AddToGroup(groupName, connectionID)
}

func (c *Hub) RemoveFromGroup(groupName, connectionID string) {
	c.Groups().RemoveFromGroup(groupName, connectionID)
}

func (c *Hub) Broadcast(groupName, target, message string) {
	c.Clients().Group(groupName).Send(target, message)
}

func (c *Hub) Echo(target, message string) {
	c.Clients().Caller().Send(target, message)
}
