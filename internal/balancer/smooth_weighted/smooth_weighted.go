package smooth_weighted

import (
	"encoding/json"
	"github.com/liuxp0827/grpc-lb/instance"
	bl "google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"strconv"
	"sync"
)

const (
	Name          = "smooth_weighted_lb"
	WeightTag     = "weight"
	defaultWeight = 0
)

func newBuilder() bl.Builder {
	return base.NewBalancerBuilderV2(Name, &smoothWeightPickerBuilder{}, base.Config{HealthCheck: true})
}

func init() {
	bl.Register(newBuilder())
}

type smoothWeightPickerBuilder struct{}

func (*smoothWeightPickerBuilder) Build(info base.PickerBuildInfo) bl.V2Picker {
	if len(info.ReadySCs) == 0 {
		return base.NewErrPickerV2(bl.ErrNoSubConnAvailable)
	}

	p := smoothWeightPicker{}

	p.weightPeers = make([]weightPeer, 0, len(info.ReadySCs))
	for sc, info := range info.ReadySCs {

		weight := defaultWeight

		switch md := info.Address.Metadata.(type) {
		case *map[string]string:
			if md != nil {
				weight = getWeight(*md)
			}
		case *instance.Metadata:
			if md != nil {
				weight = getWeight(*md)
			}
		case string:
			_md := loadMetadata(md)
			weight = getWeight(_md)
		}

		wp := weightPeer{
			subConn: sc,
			weight:  weight,
		}

		p.weightPeers = append(p.weightPeers, wp)
	}

	return &p
}

type weightPeer struct {
	subConn         bl.SubConn
	weight          int
	effectiveWeight int
	currentWeight   int
}

type smoothWeightPicker struct {
	weightPeers []weightPeer
	mu          sync.Mutex
}

func (p *smoothWeightPicker) Pick(bl.PickInfo) (bl.PickResult, error) {
	if len(p.weightPeers) == 1 { // 如果只有一个peer，直接返回，避免锁竞争
		return bl.PickResult{SubConn: p.weightPeers[0].subConn}, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	best := -1
	total := 0
	for i := 0; i < len(p.weightPeers); i++ {
		wp := &p.weightPeers[i]

		wp.currentWeight += wp.effectiveWeight
		total += wp.effectiveWeight
		if wp.effectiveWeight < wp.weight {
			wp.effectiveWeight++
		}
		if best == -1 || wp.currentWeight > p.weightPeers[best].currentWeight {
			best = i
		}
	}
	p.weightPeers[best].currentWeight -= total

	return bl.PickResult{SubConn: p.weightPeers[best].subConn}, nil
}

func loadMetadata(md string) map[string]string {
	m := map[string]string{}
	json.Unmarshal([]byte(md), &m)
	return m
}

func getWeight(md map[string]string) int {
	w, ok := md[WeightTag]
	if !ok {
		return defaultWeight
	}
	weight, err := strconv.Atoi(w)
	if err == nil {
		return weight
	}
	return defaultWeight
}
