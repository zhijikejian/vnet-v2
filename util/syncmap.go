package util

import "sync"

// 自定义的 类sync.Map的线程安全map，并发安全，加锁性能损失，增加几个方法
type SyncMap struct {
	sync.RWMutex             // 读写锁
	m            map[any]any // 映射
}

func NewSyncMap() SyncMap {
	return SyncMap{m: make(map[any]any)}
}
func (m *SyncMap) Load(key any) (any, bool) {
	m.RLock()
	value, ok := m.m[key]
	m.RUnlock()
	return value, ok
}
func (m *SyncMap) Store(key any, value any) {
	m.Lock()
	m.m[key] = value
	m.Unlock()
}
func (m *SyncMap) Delete(key any) {
	m.Lock()
	delete(m.m, key)
	m.Unlock()
}
func (m *SyncMap) Clear() {
	m.Lock()
	for k := range m.m {
		delete(m.m, k)
	}
	m.Unlock()
}
func (m *SyncMap) Len() int {
	m.RLock()
	var length = len(m.m)
	m.RUnlock()
	return length
}
func (m *SyncMap) Keys() []any {
	var keys = []any{}
	m.RLock()
	for k := range m.m {
		keys = append(keys, k)
	}
	m.RUnlock()
	return keys
}
func (m *SyncMap) Values() []any {
	var values = []any{}
	m.RLock()
	for _, v := range m.m {
		values = append(values, v)
	}
	m.RUnlock()
	return values
}
func (m *SyncMap) KVs() ([]any, []any) {
	var keys = []any{}
	var values = []any{}
	m.RLock()
	for k, v := range m.m {
		keys = append(keys, k)
		values = append(values, v)
	}
	m.RUnlock()
	return keys, values
}
func (m *SyncMap) KVlist() [][]any {
	var kvlist = [][]any{}
	m.RLock()
	for k, v := range m.m {
		kv := []any{k, v}
		kvlist = append(kvlist, kv)
	}
	m.RUnlock()
	return kvlist
}
func (m *SyncMap) Range(f func(k, v any) bool) {
	var mm = make(map[any]any)
	m.RLock()
	for k, v := range m.m { // 深复制
		mm[k] = v
	}
	m.RUnlock()
	for k, v := range mm {
		if !f(k, v) {
			break
		}
	}
}
func (m *SyncMap) From(fm map[any]any) {
	// m.Clear()
	m.Lock()
	for k := range m.m {
		delete(m.m, k)
	}
	for k, v := range fm {
		m.m[k] = v
	}
	m.Unlock()
}
func (m *SyncMap) From_string_string(fm map[string]string) {
	m.Lock()
	for k := range m.m {
		delete(m.m, k)
	}
	for k, v := range fm {
		m.m[k] = v
	}
	m.Unlock()
}
func (m *SyncMap) From_string_serve(fm map[string]Serve) {
	m.Lock()
	for k := range m.m {
		delete(m.m, k)
	}
	for k, v := range fm {
		m.m[k] = v
	}
	m.Unlock()
}
func (m *SyncMap) From_string_use(fm map[string]Use) {
	m.Lock()
	for k := range m.m {
		delete(m.m, k)
	}
	for k, v := range fm {
		m.m[k] = v
	}
	m.Unlock()
}
func (m *SyncMap) From_string_map_string_serve(fm map[string]map[string]Serve) {
	m.Lock()
	for k := range m.m {
		delete(m.m, k)
	}
	for k, v := range fm {
		m.m[k] = v
	}
	m.Unlock()
}
