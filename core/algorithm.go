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
	"math"
	"sort"
	"sync"
	"sync/atomic"
)

const (
	// The format of the host replica name
	hostReplicaFormat = `%s%d`
)

var (
	// the default number of replicas
	defaultReplicaNum = 10

	// the load bound factor
	// ref: https://research.googleblog.com/2017/04/consistent-hashing-with-bounded-loads.html
	loadBoundFactor = 0.25

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

	// the total loads of all replicas
	totalLoad int64

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
		totalLoad:          0,
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

// UpdateLoad Sets the load of `host` to the given `load`
func (c *Consistent) UpdateLoad(host string, load int64) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.hostMap[host]; !ok {
		return
	}
	c.totalLoad = c.totalLoad - c.hostMap[host].LoadBound + load
	c.hostMap[host].LoadBound = load
}

// Hosts Return the list of real hosts
func (c *Consistent) Hosts() []string {
	c.RLock()
	defer c.RUnlock()

	hosts := make([]string, 0)
	for k := range c.hostMap {
		hosts = append(hosts, k)
	}
	return hosts
}

func (c *Consistent) GetKey(key string) (string, error) {
	hashedKey := c.hashFunc(key)
	idx := c.searchKey(hashedKey)
	return c.replicaHostMap[c.sortedHostsHashSet[idx]], nil
}

// GetKeyLeast It uses consistent-hashing With Bounded loads to pick the least loaded host that can serve the key
//
// It returns ErrNoHosts if the ring has no hosts in it.
//
// ref: https://research.googleblog.com/2017/04/consistent-hashing-with-bounded-loads.html
func (c *Consistent) GetKeyLeast(key string) (string, error) {
	c.RLock()
	defer c.RUnlock()

	if len(c.replicaHostMap) == 0 {
		return "", ErrHostNotFound
	}

	hashedKey := c.hashFunc(key)
	idx := c.searchKey(hashedKey) // Find the first host that may serve the key

	i := idx
	for {
		host := c.replicaHostMap[c.sortedHostsHashSet[i]]
		loadChecked, err := c.checkLoadCapacity(host)
		if err != nil {
			return "", err
		}
		if loadChecked {
			return host, nil
		}
		i++

		// if idx goes to the end of the ring, start from the beginning
		if i >= len(c.replicaHostMap) {
			i = 0
		}
	}
}

// Inc Increments the load of host by 1
//
// should only be used with if you obtained a host with GetLeast
func (c *Consistent) Inc(hostName string) {
	c.Lock()
	defer c.Unlock()

	atomic.AddInt64(&c.hostMap[hostName].LoadBound, 1)
	atomic.AddInt64(&c.totalLoad, 1)
}

// Done Decrements the load of host by 1
//
// should only be used with if you obtained a host with GetLeast
func (c *Consistent) Done(host string) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.hostMap[host]; !ok {
		return
	}
	atomic.AddInt64(&c.hostMap[host].LoadBound, -1)
	atomic.AddInt64(&c.totalLoad, -1)
}

// GetLoads Returns the loads of all the hosts
func (c *Consistent) GetLoads() map[string]int64 {
	c.RLock()
	defer c.RUnlock()

	loads := make(map[string]int64)
	for k, v := range c.hostMap {
		loads[k] = atomic.LoadInt64(&v.LoadBound)
	}
	return loads
}

// MaxLoad Returns the maximum load of the single host
// which is:
// (total_load/number_of_hosts)*1.25
// total_load is the total number of active requests served by hosts
// for more info:
// 	https://research.googleblog.com/2017/04/consistent-hashing-with-bounded-loads.html
func (c *Consistent) MaxLoad() int64 {
	if c.totalLoad == 0 {
		c.totalLoad = 1
	}

	var avgLoadPerNode float64
	avgLoadPerNode = float64(c.totalLoad / int64(len(c.hostMap)))
	if avgLoadPerNode == 0 {
		avgLoadPerNode = 1
	}
	avgLoadPerNode = math.Ceil(avgLoadPerNode * (1 + loadBoundFactor))
	return int64(avgLoadPerNode)
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

// checkLoadCapacity check if the host can serve the key within load bound
func (c *Consistent) checkLoadCapacity(host string) (bool, error) {

	// a safety check if someone performed c.Done more than needed
	if c.totalLoad < 0 {
		c.totalLoad = 0
	}

	var avgLoadPerNode float64
	avgLoadPerNode = float64((c.totalLoad + 1) / int64(len(c.hostMap)))
	if avgLoadPerNode == 0 {
		avgLoadPerNode = 1
	}
	avgLoadPerNode = math.Ceil(avgLoadPerNode * (1 + loadBoundFactor))

	candidateHost, ok := c.hostMap[host]
	if !ok {
		return false, ErrHostNotFound
	}

	if float64(candidateHost.LoadBound)+1 <= avgLoadPerNode {
		return true, nil
	}

	return false, nil
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
