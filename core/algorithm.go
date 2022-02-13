// Package core
//
// An implementation of Consistent Hashing Algorithm described in Golang
//
// ref: https://en.wikipedia.org/wiki/Consistent_hashing
package core

import (
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"
)

const (
	// The format of the host replica name
	hostReplicaFormat = `%s%d`
)

var (
	// the default number of replicas
	defaultReplicaNum = 10

	// the default Hash function for keys
	defaultHashFunc = func(key string) uint64 {
		out := sha512.Sum512([]byte(key))
		return binary.LittleEndian.Uint64(out[:])
	}
)

// Consistent is an implementation of consistent-hashing-algorithm
type Consistent struct {
	// the number of replicas
	replicaNum int

	// the hash function for keys
	hashFunc func(key string) uint64

	// the map of virtual nodes	to hosts
	hostMap map[string]*Host

	// the map of hashed virtual nodes to host name
	replicaHostMap map[uint64]string

	// the hash ring
	sortedHostsHashSet []uint64

	// the hash ring lock
	sync.RWMutex
}

func NewConsistent(replicaNum int, hashFunc func(key string) uint64) *Consistent {
	if replicaNum <= 0 {
		replicaNum = defaultReplicaNum
	}

	if hashFunc == nil {
		hashFunc = defaultHashFunc
	}

	return &Consistent{
		replicaNum:         replicaNum,
		hashFunc:           hashFunc,
		hostMap:            make(map[string]*Host),
		replicaHostMap:     make(map[uint64]string),
		sortedHostsHashSet: make([]uint64, 0),
	}
}

func (c *Consistent) RegisterHost(hostName string) error {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.hostMap[hostName]; ok {
		return ErrHostAlreadyExists
	}

	c.hostMap[hostName] = &Host{
		Name:      hostName,
		LoadBound: 0,
	}

	for i := 0; i < c.replicaNum; i++ {
		hashedIdx := c.hashFunc(fmt.Sprintf(hostReplicaFormat, hostName, i))
		c.replicaHostMap[hashedIdx] = hostName
		c.sortedHostsHashSet = append(c.sortedHostsHashSet, hashedIdx)
	}

	// sort hashes in ascending order
	sort.Slice(c.sortedHostsHashSet, func(i int, j int) bool {
		if c.sortedHostsHashSet[i] < c.sortedHostsHashSet[j] {
			return true
		}
		return false
	})

	return nil
}

func (c *Consistent) UnregisterHost(hostName string) error {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.hostMap[hostName]; !ok {
		return ErrHostNotFound
	}

	delete(c.hostMap, hostName)

	for i := 0; i < c.replicaNum; i++ {
		hashedIdx := c.hashFunc(fmt.Sprintf(hostReplicaFormat, hostName, i))
		delete(c.replicaHostMap, hashedIdx)
		c.delHashIndex(hashedIdx)
	}

	return nil
}

// Hosts Return the list of real hosts
func (c *Consistent) Hosts() []string {
	c.RLock()
	defer c.RUnlock()

	hosts := make([]string, 0)
	for k, _ := range c.hostMap {
		hosts = append(hosts, k)
	}
	return hosts
}

func (c *Consistent) GetKey(key string) (string, error) {
	hashedKey := c.hashFunc(key)
	idx := c.searchKey(hashedKey)
	return c.replicaHostMap[c.sortedHostsHashSet[idx]], nil
}

func (c *Consistent) searchKey(key uint64) int {
	idx := sort.Search(len(c.sortedHostsHashSet), func(i int) bool {
		return c.sortedHostsHashSet[i] >= key
	})

	if idx >= len(c.sortedHostsHashSet) {
		// make search as a ring
		idx = 0
	}

	return idx
}

// Remove hashed host index from the hash ring
func (c *Consistent) delHashIndex(val uint64) {
	idx := -1
	l := 0
	r := len(c.sortedHostsHashSet) - 1
	for l <= r {
		m := (l + r) / 2
		if c.sortedHostsHashSet[m] == val {
			idx = m
			break
		} else if c.sortedHostsHashSet[m] < val {
			l = m + 1
		} else if c.sortedHostsHashSet[m] > val {
			r = m - 1
		}
	}
	if idx != -1 {
		c.sortedHostsHashSet = append(c.sortedHostsHashSet[:idx], c.sortedHostsHashSet[idx+1:]...)
	}
}
