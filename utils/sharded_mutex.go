package utils

import (
	"sync"
)

const shardedMutexCount = 64

type ShardedMutex [shardedMutexCount]sync.Mutex

func (sm *ShardedMutex) Lock(v uint) {
	(*sm)[v%shardedMutexCount].Lock()
}

func (sm *ShardedMutex) Unlock(v uint) {
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
