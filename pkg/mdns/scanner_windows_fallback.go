//go:build windows

package mdns

import (
	"context"
	"sync"
)

// browseMulticastAndQU runs the pure-Go multicast backend and the QU
// unicast-response backend in parallel, merging deduplicated results.
// This is the fallback path used on Windows < build 1703 (32-bit) or when
// the Windows DNS Service API is unavailable.
func (s *Scanner) browseMulticastAndQU(ctx context.Context) (<-chan ServiceInstance, error) {
	quCh, err := s.browseWithQU(ctx)
	if err != nil {
		return nil, err
	}

	mcCh, mcErr := s.browseWithPureGo(ctx)
	if mcErr != nil {
		s.opts.Logger.Printf("mdns: multicast backend unavailable (%v); using QU only", mcErr)
		return quCh, nil
	}

	s.debugf("mdns: running multicast and QU backends in parallel")

	out := make(chan ServiceInstance)
	seen := make(map[string]bool)
	var mu sync.Mutex

	forward := func(src <-chan ServiceInstance, wg *sync.WaitGroup) {
		defer wg.Done()
		for inst := range src {
			mu.Lock()
			dup := seen[inst.Name]
			seen[inst.Name] = true
			mu.Unlock()
			if dup {
				continue
			}
			select {
			case out <- inst:
			case <-ctx.Done():
				return
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go forward(mcCh, &wg)
	go forward(quCh, &wg)

	go func() {
		wg.Wait()
		close(out)
	}()

	return out, nil
}
