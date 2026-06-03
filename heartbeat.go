package main

import (
	"net/http"
	"sync"
	"time"
)

// =============================================================================
// Heartbeat Monitor — Periodic connectivity check with exponential backoff
// =============================================================================

const (
	heartbeatFailureThreshold = 2 // consecutive failures before declaring connection lost
	heartbeatMaxRetries       = 5 // max exponential backoff steps
	heartbeatRequestTimeout   = 5 * time.Second
	heartbeatDefaultInterval  = 45 * time.Second
)

// heartbeatRetryDelays defines the exponential backoff sequence.
var heartbeatRetryDelays = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	16 * time.Second,
}

// HeartbeatState represents the current connection state.
type HeartbeatState int

const (
	HeartbeatRunning  HeartbeatState = 0
	HeartbeatRetrying HeartbeatState = 1
	HeartbeatLost     HeartbeatState = 2
)

// Heartbeat monitors network connectivity via periodic HTTP HEAD requests.
type Heartbeat struct {
	mu sync.Mutex

	client   *http.Client
	interval time.Duration
	urls     []string
	urlIndex int

	running  bool
	state    HeartbeatState
	isOnline bool

	failCount  int
	retryCount int
	retryDelay time.Duration

	ticker *time.Ticker
	done   chan struct{}

	// Callbacks
	onConnected func()
	onLost      func()
	onReconnect func()
}

// NewHeartbeat creates a new Heartbeat monitor.
func NewHeartbeat() *Heartbeat {
	return &Heartbeat{
		client: &http.Client{
			Timeout: heartbeatRequestTimeout,
		},
		interval: heartbeatDefaultInterval,
		urls:     []string{"https://www.baidu.com", "https://www.bing.com"},
		done:     make(chan struct{}),
		isOnline: true, // assume connected at start
	}
}

// Start begins periodic heartbeat checks.
func (hb *Heartbeat) Start() {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	if hb.running {
		return
	}

	hb.running = true
	hb.ticker = time.NewTicker(hb.interval)
	hb.state = HeartbeatRunning

	go hb.loop()

	GetLogger().Info("Heartbeat started (interval: %v)", hb.interval)
}

// Stop halts the heartbeat monitor.
func (hb *Heartbeat) Stop() {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	if !hb.running {
		return
	}

	hb.running = false
	if hb.ticker != nil {
		hb.ticker.Stop()
	}
	close(hb.done)
	GetLogger().Info("Heartbeat stopped")
}

// IsConnected returns whether the network is currently considered connected.
func (hb *Heartbeat) IsConnected() bool {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	return hb.isOnline
}

// SetInterval changes the check interval (only takes effect after next tick).
func (hb *Heartbeat) SetInterval(seconds int) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	if seconds < 15 {
		seconds = 15
	}
	if seconds > 300 {
		seconds = 300
	}
	hb.interval = time.Duration(seconds) * time.Second
	if hb.ticker != nil {
		hb.ticker.Reset(hb.interval)
	}
}

// SetURLs updates the check URLs.
func (hb *Heartbeat) SetURLs(urls []string) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	if len(urls) > 0 {
		hb.urls = urls
		hb.urlIndex = 0
	}
}

// OnConnected sets the callback for when connection is restored.
func (hb *Heartbeat) OnConnected(fn func()) { hb.onConnected = fn }

// OnLost sets the callback for when connection is lost.
func (hb *Heartbeat) OnLost(fn func()) { hb.onLost = fn }

// OnReconnectRequested sets the callback for when a reconnect should be triggered.
func (hb *Heartbeat) OnReconnectRequested(fn func()) { hb.onReconnect = fn }

// ForceReconnect triggers an immediate reconnect sequence.
func (hb *Heartbeat) ForceReconnect() {
	hb.mu.Lock()
	hb.running = false
	if hb.ticker != nil {
		hb.ticker.Stop()
	}
	hb.state = HeartbeatRetrying
	hb.retryCount = 0
	hb.mu.Unlock()

	GetLogger().Info("Force reconnect triggered")
	hb.startRetrySequence()
}

func (hb *Heartbeat) loop() {
	for {
		select {
		case <-hb.done:
			return
		case <-hb.ticker.C:
			hb.check()
		}
	}
}

func (hb *Heartbeat) check() {
	hb.mu.Lock()
	if len(hb.urls) == 0 {
		hb.mu.Unlock()
		return
	}
	url := hb.urls[hb.urlIndex]
	hb.urlIndex = (hb.urlIndex + 1) % len(hb.urls)
	hb.mu.Unlock()

	GetLogger().Debug("Heartbeat: HEAD %s", url)

	resp, err := hb.client.Head(url)
	if err != nil {
		GetLogger().Warn("Heartbeat failed: %v", err)
		hb.onFailure()
		return
	}
	resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		hb.onSuccess()
	} else {
		GetLogger().Warn("Heartbeat got status %d from %s", resp.StatusCode, url)
		hb.onFailure()
	}
}

func (hb *Heartbeat) onSuccess() {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	hb.failCount = 0

	if hb.state == HeartbeatLost || hb.state == HeartbeatRetrying {
		wasOffline := !hb.isOnline
		hb.isOnline = true
		hb.state = HeartbeatRunning
		hb.retryCount = 0
		hb.retryDelay = 0

		if wasOffline && hb.onReconnect != nil {
			GetLogger().Info("Heartbeat: connection restored")
			go hb.onReconnect()
		}
	}
}

func (hb *Heartbeat) onFailure() {
	hb.mu.Lock()
	hb.failCount++
	wasOnline := hb.isOnline

	if hb.failCount >= heartbeatFailureThreshold && hb.isOnline {
		hb.isOnline = false
		hb.state = HeartbeatLost
		GetLogger().Warn("Heartbeat: connection LOST (%d consecutive failures)", hb.failCount)
		if hb.onLost != nil {
			go hb.onLost()
		}
	} else if hb.failCount >= heartbeatFailureThreshold && !hb.isOnline {
		hb.state = HeartbeatLost
	}

	if wasOnline && !hb.isOnline {
		// Start retry sequence
		hb.mu.Unlock()
		hb.startRetrySequence()
		return
	}
	hb.mu.Unlock()
}

func (hb *Heartbeat) startRetrySequence() {
	hb.mu.Lock()
	if hb.retryCount >= heartbeatMaxRetries {
		hb.mu.Unlock()
		GetLogger().Warn("Heartbeat: max retries reached, giving up")
		return
	}

	delay := heartbeatRetryDelays[hb.retryCount]
	hb.retryCount++
	hb.state = HeartbeatRetrying
	hb.mu.Unlock()

	GetLogger().Info("Heartbeat: retry %d/%d in %v", hb.retryCount, heartbeatMaxRetries, delay)

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-hb.done:
		return
	case <-timer.C:
	}

	// Check one URL
	hb.mu.Lock()
	if len(hb.urls) == 0 {
		hb.mu.Unlock()
		return
	}
	url := hb.urls[0]
	hb.mu.Unlock()

	resp, err := hb.client.Head(url)
	if err != nil {
		hb.onFailure()
		hb.startRetrySequence()
		return
	}
	resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		hb.onSuccess()
	} else {
		hb.onFailure()
		hb.startRetrySequence()
	}
}
