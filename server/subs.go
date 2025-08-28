package server

import "sync"

type submanager struct {
	connections *sync.Map
	subscribers *sync.Map
	monitors    *sync.Map
}

func NewSubManager() *submanager {
	return &submanager{
		connections: &sync.Map{},
		subscribers: &sync.Map{},
		monitors:    &sync.Map{},
	}
}

func (sm *submanager) AddConnection(id string) {
	sm.connections.Store(id, "")
}

func (sm *submanager) RemoveConnection(id string) {
	sm.connections.Delete(id)
	sm.RemoveSubscriber(id)
}

func (sm *submanager) ListConnections() []string {
	var ids []string
	sm.connections.Range(func(key, value interface{}) bool {
		ids = append(ids, key.(string))
		return true
	})
	return ids
}

func (sm *submanager) AddSubscriber(channel, id string) {
	csubs, ok := sm.subscribers.Load(channel)
	if !ok {
		csubs = &sync.Map{}
		csubs.(*sync.Map).Store(id, "")
		sm.subscribers.Store(channel, csubs)
	}

	csubs.(*sync.Map).Store(id, "")
}

func (sm *submanager) RemoveSubscriber(id string) {
	sm.subscribers.Range(func(key any, value any) bool {
		m := value.(*sync.Map)
		m.Delete(id)
		return true
	})
}

func (sm *submanager) ListSubscribers(channel string) []string {
	subs, _ := sm.subscribers.Load(channel)
	if subs == nil {
		return nil
	}

	var ids []string
	subs.(*sync.Map).Range(func(key, value interface{}) bool {
		ids = append(ids, key.(string))
		return true
	})

	return ids
}

// Monitor
func (sm *submanager) AddMonitor(channel, id string) {
	csubs, ok := sm.monitors.Load(channel)
	if !ok {
		csubs = &sync.Map{}
		csubs.(*sync.Map).Store(id, "")
		sm.monitors.Store(channel, csubs)
	}

	csubs.(*sync.Map).Store(id, "")
}

func (sm *submanager) RemoveMonitor(id string) {
	sm.monitors.Range(func(key any, value any) bool {
		m := value.(*sync.Map)
		m.Delete(id)
		return true
	})
}

func (sm *submanager) ListMontitors(channel string) []string {
	subs, _ := sm.monitors.Load(channel)
	if subs == nil {
		return nil
	}

	var ids []string
	subs.(*sync.Map).Range(func(key, value interface{}) bool {
		ids = append(ids, key.(string))
		return true
	})

	return ids
}
