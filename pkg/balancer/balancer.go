package balancer

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"google.golang.org/grpc/resolver"
)

const (
	// EtcdDialTimeout - default timeout for dialling the etcd
	EtcdDialTimeout = 5
	// EtcdRequestTimeout - default timeout for requesting the etcd
	EtcdRequestTimeout = 5
	// EtcdRegisterTTL - default register ttl for service
	EtcdRegisterTTL = 30
	// TimerCheckInterval - default interval time for checking if service is deleted
	TimerCheckInterval = 15
)

var (
	scheme = "services"
)

// EtcdBalancer defines a balancer based on etcd
type EtcdBalancer struct {
	client      *clientv3.Client // the etcd client
	resolver    resolver.Builder // the etcd resolver
	servicePath string           // the service path
	done        chan struct{}    // notify exit
}

// NewEtcdBalancer returns a etcd balancer
func NewEtcdBalancer(addr string) *EtcdBalancer {
	// new a etcd client which based on grpc protocol
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   strings.Split(addr, ";"),
		DialTimeout: time.Second * EtcdDialTimeout,
	})
	if err != nil {
		panic(err)
	}

	// new a etcd resolver
	resolver := newResolver(client)

	return &EtcdBalancer{
		client:   client,
		resolver: resolver,
		done:     make(chan struct{}),
	}
}

// Resolver returns a etcd resolver
func (s *EtcdBalancer) Resolver() resolver.Builder {
	return s.resolver
}

// Endpoint for service
type Endpoint struct {
	IP       string            `json:"ip"`
	Port     string            `json:"port"`
	Protocol string            `json:"protocol"`
	Version  string            `json:"version"`
	Metadata map[string]string `json:"metadata"`
}

// Service structure for registering
type Service struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Endpoints []Endpoint `json:"endpoints"`
}

func (s *EtcdBalancer) register(service *Service) error {
	ctx := context.Background()
	// try to get the specific service instance information from etcd registry
	res, err := s.client.Get(ctx, s.servicePath)
	if err != nil {
		return err
	}
	if res.Count == 0 {
		// if there is no service, try to register service to etcd registry
		if err := s.keepalive(ctx, service); err != nil {
			return err
		}
	}
	return nil
}

// Register service with service path to registry
func (s *EtcdBalancer) Register(wg *sync.WaitGroup, service *Service) error {
	defer wg.Done()

	// init the service path
	s.servicePath = "/" + scheme + "/" + service.Name + "/" + service.ID

	// register once before starting the timer
	if err := s.register(service); err != nil {
		panic(err)
	}

	// start a timer for register and keep alive the service
	ticker := time.NewTimer(time.Second * TimerCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return nil
		case <-ticker.C:
			// register the service periodly
			if err := s.register(service); err != nil {
				log.Printf("register service {%v}: %v", service, err)
			}
			ticker.Reset(time.Second * TimerCheckInterval)
		}
	}
}

func (s *EtcdBalancer) keepalive(ctx context.Context, service *Service) error {
	// Grant creates a new lease.
	lease, err := s.client.Grant(ctx, EtcdRegisterTTL)
	if err != nil {
		return err
	}

	body, err := json.Marshal(service)
	if err != nil {
		return err
	}
	// put the service into etcd registry
	res, err := s.client.Put(ctx, s.servicePath, string(body), clientv3.WithLease(lease.ID))
	if err != nil {
		return err
	}
	log.Printf("Put service {%s}, revision: {%v}", s.servicePath, res.Header.Revision)

	// KeepAlive keeps the given lease alive forever
	if _, err := s.client.KeepAlive(ctx, lease.ID); err != nil {
		return err
	}

	return nil
}

// UnRegister service with service path from etcd registry
func (s *EtcdBalancer) UnRegister() error {
	if s.servicePath == "" {
		return errors.New("service path is empty")
	}

	res, err := s.client.Delete(context.Background(), s.servicePath)
	if err != nil {
		return err
	}
	log.Printf("service {%s} is deleted -> %v", s.servicePath, (res.Deleted == 1))

	return nil
}

// Close the etcd balancer gracefully
func (s *EtcdBalancer) Close() {
	// close the timer first
	if s.done != nil {
		close(s.done)
	}

	// close the etcd client
	if s.client != nil {
		s.client.Close()
	}
}
