package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"google.golang.org/grpc/resolver"
)

const scheme = "etcd"

// BuilderAndResolver implements interfaces 'Builder and Resolver'
type BuilderAndResolver struct {
	client *clientv3.Client // the etcd client

	// resolver.ClientConn contains the callbacks for resolver to notify any updates to the gRPC ClientConn.
	cc resolver.ClientConn

	done chan struct{} // close the resolver
}

// newResolver returns a etcd resolver
func newResolver(client *clientv3.Client) resolver.Builder {
	return &BuilderAndResolver{
		client: client,
		done:   make(chan struct{}),
	}
}

/** interfaces implementation for resolver.Builder **/

// Build creates a new resolver for the given target.
func (s *BuilderAndResolver) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	// grpc client connection
	s.cc = cc

	// start a goroutine for watching the service path
	if err := s.watch(target); err != nil {
		return nil, err
	}

	return s, nil
}

// Scheme returns the scheme supported by this resolver.
func (s *BuilderAndResolver) Scheme() string {
	return scheme
}

/** interfaces implementation for resolver.Resolver **/

// ResolveNow will be called by gRPC to try to resolve the target name again.
func (s *BuilderAndResolver) ResolveNow(o resolver.ResolveNowOptions) {}

// Close the resolver.
func (s *BuilderAndResolver) Close() {
	if s.done != nil {
		close(s.done)
	}
}

// watch and handle the address changes for service from etcd registry
func (s *BuilderAndResolver) watch(target resolver.Target) error {
	prefix := "/" + prefix + "/" + target.Endpoint
	// get the root directory of the service
	root, err := s.client.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		log.Printf("client Get {%s}: %v", prefix, err)
		return err
	}

	var addrs []resolver.Address
	// init the address from etcd registry for the service
	for _, item := range root.Kvs {
		key := string(item.Key)
		res, err := s.client.Get(context.Background(), key)
		if err != nil {
			log.Printf("client Get {%s}: %v", key, err)
			return err
		}
		for _, node := range res.Kvs {
			var service Service
			if err := json.Unmarshal(node.Value, &service); err != nil {
				log.Printf("unmarshal {%q}: %v", node.Value, err)
				return err
			}
			for _, endpoint := range service.Endpoints {
				addrs = append(addrs, resolver.Address{
					Addr: fmt.Sprintf("%s:%s", endpoint.IP, endpoint.Port),
				})
			}
		}
	}

	// trigger the grpc client connection to update the addresses
	s.cc.UpdateState(resolver.State{Addresses: addrs})

	go func() {
		// watch and handle the changes for the service from etcd registry
		watchChan := s.client.Watch(context.Background(), prefix, clientv3.WithPrefix(), clientv3.WithPrevKV())

		for {
			select {
			case <-s.done:
				return

			case data := <-watchChan:
				select {
				case <-s.done:
					return
				default:
				}

				if data.Err() != nil {
					log.Printf("watch error: %v", data.Err())
					time.Sleep(time.Second * 5)
					continue
				}

				// handle the watch events
				for _, event := range data.Events {
					switch event.Type {
					case mvccpb.PUT:
						// for the addition event, the kv should not be empty
						if event.Kv == nil {
							log.Printf("current kv is nil for addition")
							continue
						}

						var service Service
						// unmarshal the json string to service
						if err := json.Unmarshal(event.Kv.Value, &service); err != nil {
							log.Printf("unmarshal {%q}: %v", event.Kv.Value, err)
							continue
						}

						var ok bool
						// handle the endpoints
						if addrs, ok = addition(addrs, service.Endpoints); ok {
							s.cc.UpdateState(resolver.State{Addresses: addrs})
						}

					case mvccpb.DELETE:
						// for the delete event, the prev kv should not be empty
						if event.PrevKv == nil {
							log.Printf("previous kv is nil for deletion")
							continue
						}

						var service Service
						// unmarshal the json string to service
						if err := json.Unmarshal(event.PrevKv.Value, &service); err != nil {
							log.Printf("unmarshal {%q}: %v", event.PrevKv.Value, err)
							continue
						}

						var ok bool
						// handle the endpoints
						if addrs, ok = deletion(addrs, service.Endpoints); ok {
							s.cc.UpdateState(resolver.State{Addresses: addrs})
						}
					}
				}
			}
		}
	}()

	return nil
}

func matchAddress(eps []Endpoint, addr string) bool {
	for _, ep := range eps {
		epAddr := fmt.Sprintf("%s:%s", ep.IP, ep.Port)
		if addr == epAddr {
			return true
		}
	}
	return false
}

func matchEndpoint(addrs []resolver.Address, ep Endpoint) bool {
	for _, item := range addrs {
		addr := fmt.Sprintf("%s:%s", ep.IP, ep.Port)
		if item.Addr == addr {
			return true
		}
	}
	return false
}

// try to add the address
func addition(addrs []resolver.Address, eps []Endpoint) (newAddrs []resolver.Address, added bool) {
	// copy the slice
	newAddrs = addrs[:]
	for _, ep := range eps {
		if !matchEndpoint(addrs, ep) {
			addr := fmt.Sprintf("%s:%s", ep.IP, ep.Port)
			newAddrs = append(newAddrs, resolver.Address{Addr: addr})
			added = true
		}
	}
	return
}

// try to delete the address
func deletion(addrs []resolver.Address, eps []Endpoint) (newAddrs []resolver.Address, deleted bool) {
	for _, item := range addrs {
		if matchAddress(eps, item.Addr) {
			deleted = true
			continue
		}
		newAddrs = append(newAddrs, item)
	}
	return
}
