package etcd

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
	prefix = "services"
)

// Register defines a register based on etcd
type Register struct {
	client      *clientv3.Client // the etcd client
	builder     resolver.Builder // the resolver builder
	servicePath string           // the service path
	done        chan struct{}    // notify exit
}

// NewRegister returns a etcd register
func NewRegister(addr string) *Register {
	// new a etcd client which based on grpc protocol
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   strings.Split(addr, ";"),
		DialTimeout: time.Second * EtcdDialTimeout,
	})
	if err != nil {
		panic(err)
	}

	return &Register{
		client:  client,
		builder: newResolver(client),
		done:    make(chan struct{}),
	}
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

func (s *Register) register(service *Service) error {
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

// Resolver returns the builder for etcd resolver
func (s *Register) Resolver() resolver.Builder {
	return s.builder
}

// Register service with service path to registry
func (s *Register) Register(wg *sync.WaitGroup, service *Service) error {
	defer wg.Done()

	// init the service path
	s.servicePath = "/" + prefix + "/" + service.Name + "/" + service.ID

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

func (s *Register) keepalive(ctx context.Context, service *Service) error {
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
func (s *Register) UnRegister() error {
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
func (s *Register) Close() {
	// close the timer first
	if s.done != nil {
		close(s.done)
	}

	// close the etcd client
	if s.client != nil {
		s.client.Close()
	}
}
