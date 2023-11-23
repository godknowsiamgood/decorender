package utils

import (
	"sync"
)

const shardedMutexCount = 64

type ShardedMutex [shardedMutexCount]sync.Mutex

func (sm *ShardedMutex) Lock(key string) {
	(*sm)[HashDJB2(key)%shardedMutexCount].Lock()
}

func (sm *ShardedMutex) LockInt(v int) {
	(*sm)[v%shardedMutexCount].Lock()
}

func (sm *ShardedMutex) Unlock(key string) {
	(*sm)[HashDJB2(key)%shardedMutexCount].Unlock()
}

func (sm *ShardedMutex) UnlockInt(v int) {
	(*sm)[v%shardedMutexCount].Unlock()
}

func HashDJB2(s string) int {
	var hash = 5381
	for i := 0; i < len(s); i++ {
		hash = ((hash << 5) + hash) + int(s[i])
	}
	return hash
}

func HashDJB2Num[T int | float64](v ...T) int {
	var hash = 5381
	for i := 0; i < len(v); i++ {
		hash = ((hash << 5) + hash) + int(v[i])
	}
	return hash
}
