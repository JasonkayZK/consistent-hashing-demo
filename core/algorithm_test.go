package core

import (
	"fmt"
	"testing"
)

func TestConsistent_RegisterHost(t *testing.T) {
	c := NewConsistent(0, nil)

	_ = c.RegisterHost("127.0.0.1:8000")
	if len(c.Hosts()) != 1 {
		t.Errorf("Expected 1 node in ring, got %d", len(c.Hosts()))
	}
	if len(c.sortedHostsHashSet) != len(c.Hosts())*defaultReplicaNum {
		t.Errorf("Expected %d node in sortedHostsHashSet, got %d", len(c.Hosts())*defaultReplicaNum, len(c.sortedHostsHashSet))
	}

	_ = c.RegisterHost("127.0.0.1:9999")
	if len(c.Hosts()) != 2 {
		t.Errorf("Expected 2 node in ring, got %d", len(c.Hosts()))
	}
	if len(c.sortedHostsHashSet) != len(c.Hosts())*defaultReplicaNum {
		t.Errorf("Expected %d node in sortedHostsHashSet, got %d", len(c.Hosts())*defaultReplicaNum, len(c.sortedHostsHashSet))
	}

	_ = c.RegisterHost("127.0.0.1:8000")
	if len(c.Hosts()) != 2 {
		t.Errorf("Expected 2 node in ring, got %d", len(c.Hosts()))
	}
	if len(c.sortedHostsHashSet) != len(c.Hosts())*defaultReplicaNum {
		t.Errorf("Expected %d node in sortedHostsHashSet, got %d", len(c.Hosts())*defaultReplicaNum, len(c.sortedHostsHashSet))
	}
}

func TestConsistent_UnregisterHost(t *testing.T) {
	c := NewConsistent(0, nil)

	_ = c.RegisterHost("127.0.0.1:8000")
	_ = c.RegisterHost("127.0.0.1:9999")
	_ = c.RegisterHost("127.0.0.1:8000")

	if len(c.Hosts()) != 2 {
		t.Errorf("Expected 2 node in ring, got %d", len(c.Hosts()))
	}

	err := c.UnregisterHost("127.0.0.1:8000")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if len(c.Hosts()) != 1 {
		t.Errorf("Expected 1 node in ring, got %d", len(c.Hosts()))
	}

	err = c.UnregisterHost("127.0.0.1:8000")
	if err == nil {
		t.Errorf("Expected error, got nil")
		if err != ErrHostNotFound {
			t.Errorf("Expected error %s, got %s", ErrHostNotFound, err)
		}
	}
	if len(c.Hosts()) != 1 {
		t.Errorf("Expected 1 node in ring, got %d", len(c.Hosts()))
	}

	err = c.UnregisterHost("127.0.0.1:8848")
	if err == nil {
		t.Errorf("Expected error, got nil")
		if err != ErrHostNotFound {
			t.Errorf("Expected error %s, got %s", ErrHostNotFound, err)
		}
	}
	if len(c.Hosts()) != 1 {
		t.Errorf("Expected 1 node in ring, got %d", len(c.Hosts()))
	}

	err = c.UnregisterHost("127.0.0.1:9999")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if len(c.Hosts()) != 0 {
		t.Errorf("Expected 0 node in ring, got %d", len(c.Hosts()))
	}
}

func TestConsistent_Hosts(t *testing.T) {
	c := NewConsistent(0, nil)

	_ = c.RegisterHost("127.0.0.1:8000")
	_ = c.RegisterHost("192.168.0.1:1234")

	for _, h := range c.Hosts() {
		fmt.Println(h)
	}
}

func TestConsistent_GetKey(t *testing.T) {
	c := NewConsistent(0, nil)
	_ = c.RegisterHost("127.0.0.1:8000")

	host, err := c.GetKey("1234")
	if err != nil {
		t.Fatal(err)
	}
	if host != "127.0.0.1:8000" {
		t.Fatal("returned host is not what expected")
	}

	_ = c.RegisterHost("192.168.0.1:8999")
	host, err = c.GetKey("23452345")
	if err != nil {
		t.Fatal(err)
	}
	if host != "192.168.0.1:8999" {
		t.Fatal("returned host is not what expected")
	}
}

func TestConsistent_GetKeyLeast(t *testing.T) {
	c := NewConsistent(0, nil)

	_ = c.RegisterHost("127.0.0.1:8000")
	_ = c.RegisterHost("92.0.0.1:8000")

	for i := 0; i < 100; i++ {
		host, err := c.GetKeyLeast("1234")
		if err != nil {
			t.Fatal(err)
		}
		c.Inc(host)
	}

	for k, v := range c.GetLoads() {
		if v > c.MaxLoad() {
			t.Fatalf("host %s is overloaded. %d > %d\n", k, v, c.MaxLoad())
		}
	}
	fmt.Println("Max load per node", c.MaxLoad())
	fmt.Println(c.GetLoads())
}

func TestConsistent_IncDone(t *testing.T) {
	c := NewConsistent(0, nil)

	_ = c.RegisterHost("127.0.0.1:8000")
	_ = c.RegisterHost("92.0.0.1:8000")

	host, err := c.GetKeyLeast("3124512")
	if err != nil {
		t.Fatal(err)
	}

	c.Inc(host)
	if c.hostMap[host].LoadBound != 1 {
		t.Fatalf("host %s load should be 1\n", host)
	}

	c.Done(host)
	if c.hostMap[host].LoadBound != 0 {
		t.Fatalf("host %s load should be 0\n", host)
	}

}

func TestConsistent_delHashIndex(t *testing.T) {
	items := []uint64{0, 1, 2, 3, 5, 20, 22, 23, 25, 27, 28, 30, 35, 37, 1008, 1009}
	deletes := []uint64{25, 37, 1009, 3, 100000}

	c := &Consistent{}
	c.sortedHostsHashSet = append(c.sortedHostsHashSet, items...)

	fmt.Printf("before deletion%+v\n", c.sortedHostsHashSet)

	for _, val := range deletes {
		c.delHashIndex(val)
	}

	for _, val := range deletes {
		for _, item := range c.sortedHostsHashSet {
			if item == val {
				t.Fatalf("%d wasn't deleted\n", val)
			}
		}
	}

	fmt.Printf("after deletions: %+v\n", c.sortedHostsHashSet)
}
