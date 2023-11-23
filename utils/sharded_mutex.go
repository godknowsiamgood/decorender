package utils

import (
	"sync"
)

const shardedMutexCount = 64

type ShardedMutex [shardedMutexCount]sync.Mutex

func (sm *ShardedMutex) Lock(key string) {
	(*sm)[HashDJB2(key)%shardedMutexCount].Lock()
}

func (sm *ShardedMutex) LockInt(v uint) {
	(*sm)[v%shardedMutexCount].Lock()
}

func (sm *ShardedMutex) Unlock(key string) {
	(*sm)[HashDJB2(key)%shardedMutexCount].Unlock()
}

func (sm *ShardedMutex) UnlockInt(v uint) {
	(*sm)[v%shardedMutexCount].Unlock()
}

func HashDJB2(s string) uint {
	var hash uint = 5381
	for i := 0; i < len(s); i++ {
		hash = ((hash << 5) + hash) + uint(s[i])
	}
	return hash
}

func HashDJB2Num[T int | float64](v ...T) uint {
	var hash uint = 5381
	for i := 0; i < len(v); i++ {
		hash = ((hash << 5) + hash) + uint(v[i])
	}
	return hash
}
