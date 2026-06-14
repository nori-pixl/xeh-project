package core

import (
	"os"
	"sync"
)

// SharedStore は全エンジン間で共有されるメモリ空間 (旧 sharedStore + storeMu)
type SharedStore struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// NewSharedStore は空のSharedStoreを生成する
func NewSharedStore() *SharedStore {
	return &SharedStore{data: make(map[string]interface{})}
}

// Set は値を1件設定する
func (s *SharedStore) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

// Get は値を1件取得する
func (s *SharedStore) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

// Snapshot は現在のデータ全体のコピーを返す (外部プロセスへの引き渡し用)
func (s *SharedStore) Snapshot() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]interface{}, len(s.data))
	for k, v := range s.data {
		out[k] = v
	}
	return out
}

// Range はロックを保持したまま全要素を読み取り専用で走査する
func (s *SharedStore) Range(f func(key string, value interface{})) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, v := range s.data {
		f(k, v)
	}
}

// ProcessList は起動した子プロセスの一覧をスレッドセーフに管理する (旧 activeProcesses + processMu)
type ProcessList struct {
	mu        sync.Mutex
	processes []*os.Process
}

// NewProcessList は空のProcessListを生成する
func NewProcessList() *ProcessList {
	return &ProcessList{}
}

// Add はプロセスを一覧に追加する
func (p *ProcessList) Add(proc *os.Process) {
	if proc == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.processes = append(p.processes, proc)
}

// KillAll は管理下の全プロセスを強制終了する (Ctrl+C時のクリーンアップ用)
func (p *ProcessList) KillAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, proc := range p.processes {
		if proc != nil {
			_ = proc.Kill()
		}
	}
}
