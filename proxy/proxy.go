package proxy

import (
	"fmt"
	"github.com/jasonkayzk/consistent-hashing-demo/core"
	"io/ioutil"
	"net/http"
)

type Proxy struct {
	consistent *core.Consistent
}

// NewProxy creates a new Proxy
func NewProxy(consistent *core.Consistent) *Proxy {
	proxy := &Proxy{
		consistent: consistent,
	}
	return proxy
}

func (p *Proxy) GetKey(key string) (string, error) {

	host, err := p.consistent.GetKey(key)
	if err != nil {
		return "", err
	}

	resp, err := http.Get(fmt.Sprintf("http://%s?key=%s", host, key))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	fmt.Printf("Response from host %s: %s\n", host, string(body))

	return string(body), nil
}

func (p *Proxy) RegisterHost(host string) error {

	err := p.consistent.RegisterHost(host)
	if err != nil {
		return err
	}

	fmt.Println(fmt.Sprintf("register host: %s success", host))
	return nil
}

func (p *Proxy) UnregisterHost(host string) error {
	err := p.consistent.UnregisterHost(host)
	if err != nil {
		return err
	}

	fmt.Println(fmt.Sprintf("unregister host: %s success", host))
	return nil
}
