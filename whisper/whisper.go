// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package whisper

import (
	"crypto/ecdsa"
	"sync"
	"time"

	"p2p-go/common"
	"p2p-go/crypto"
	"p2p-go/crypto/ecies"
	"p2p-go/filter"
	"p2p-go/logger"
	"p2p-go/logger/glog"

	"p2p-go/server"

	"gopkg.in/fatih/set.v0"
)

const (
	statusCode   = 0x00
	messagesCode = 0x01

	protocolVersion uint64 = 0x02
	protocolName           = "shh"

	signatureFlag   = byte(1 << 7)
	signatureLength = 65

	expirationCycle   = 800 * time.Millisecond
	transmissionCycle = 300 * time.Millisecond
)

const (
	DefaultTTL = 50 * time.Second
	DefaultPoW = 50 * time.Millisecond
)

type MessageEvent struct {
	To      *ecdsa.PrivateKey
	From    *ecdsa.PublicKey
	Message *Message
}

// Whisper represents a dark communication interface through the
// network, using its very own P2P communication layer.
type Whisper struct {
	protocol server.Protocol
	filters  *filter.Filters

	keys map[string]*ecdsa.PrivateKey

	messages    map[common.Hash]*Envelope // Pool of messages currently tracked by this node
	expirations map[uint32]*set.SetNonTS  // Message expiration pool (TODO: something lighter)
	poolMu      sync.RWMutex              // Mutex to sync the message and expiration pools

	peers  map[*peer]struct{} // Set of currently active peers
	peerMu sync.RWMutex       // Mutex to sync the active peer set

	quit chan struct{}
}

// New creates a Whisper client ready to communicate through the P2P
// network.
func New() *Whisper {
	whisper := &Whisper{
		filters:     filter.New(),
		keys:        make(map[string]*ecdsa.PrivateKey),
		messages:    make(map[common.Hash]*Envelope),
		expirations: make(map[uint32]*set.SetNonTS),
		peers:       make(map[*peer]struct{}),
		quit:        make(chan struct{}),
	}
	whisper.filters.Start()

	// server whisper sub protocol handler
	whisper.protocol = server.Protocol{
		Name:    protocolName,
		Version: uint(protocolVersion),
		Length:  2,
		Run:     whisper.handlePeer,
	}

	return whisper
}

// Protocol returns the whisper sub-protocol handler for this particular client.
func (w *Whisper) Protocol() server.Protocol {
	return w.protocol
}

// Version returns the whisper sub-protocols version number.
func (w *Whisper) Version() uint {
	return w.protocol.Version
}

// NewIdentity generates a new cryptographic identity for the client, and injects
// it into the known identities for message decryption.
func (w *Whisper) NewIdentity() *ecdsa.PrivateKey {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	w.keys[string(crypto.FromECDSAPub(&key.PublicKey))] = key

	return key
}

func (w *Whisper) InjectIdentity(key *ecdsa.PrivateKey) {
	w.keys[string(crypto.FromECDSAPub(&(key.PublicKey)))] = key
}

// HasIdentity checks if the the whisper node is configured with the private key
// of the specified public pair.
func (w *Whisper) HasIdentity(key *ecdsa.PublicKey) bool {
	return w.keys[string(crypto.FromECDSAPub(key))] != nil
}

// GetIdentity retrieves the private key of the specified public identity.
func (w *Whisper) GetIdentity(key *ecdsa.PublicKey) *ecdsa.PrivateKey {
	return w.keys[string(crypto.FromECDSAPub(key))]
}

// Watch installs a new message handler to run in case a matching packet arrives
// from the whisper network.
func (w *Whisper) Watch(options Filter) int {
	filter := filterer{
		to:      string(crypto.FromECDSAPub(options.To)),
		from:    string(crypto.FromECDSAPub(options.From)),
		matcher: newTopicMatcher(options.Topics...),
		fn: func(data interface{}) {
			options.Fn(data.(*Message))
		},
	}
	return w.filters.Install(filter)
}

// Unwatch removes an installed message handler.
func (w *Whisper) Unwatch(id int) {
	w.filters.Uninstall(id)
}

// Send injects a message into the whisper send queue, to be distributed in the
// network in the coming cycles.
func (w *Whisper) Send(envelope *Envelope) error {
	return w.add(envelope)
}

func (w *Whisper) Start() {
	glog.V(logger.Info).Infoln("Whisper started")
	go w.update()
}

func (w *Whisper) Stop() {
	close(w.quit)
	glog.V(logger.Info).Infoln("Whisper stopped")
}

// Messages retrieves all the currently pooled messages matching a filter id.
func (w *Whisper) Messages(id int) []*Message {
	messages := make([]*Message, 0)
	if filter := w.filters.Get(id); filter != nil {
		for _, envelope := range w.messages {
			if message := w.open(envelope); message != nil {
				if w.filters.Match(filter, createFilter(message, envelope.Topics)) {
					messages = append(messages, message)
				}
			}
		}
	}
	return messages
}

// handlePeer is called by the underlying P2P layer when the whisper sub-protocol
// connection is negotiated.
func (w *Whisper) handlePeer(peer *server.Peer, rw server.MsgReadWriter) error {
	// Create the new peer and start tracking it
	whisperPeer := newPeer(w, peer, rw)

	w.peerMu.Lock()
	w.peers[whisperPeer] = struct{}{}
	w.peerMu.Unlock()

	defer func() {
		w.peerMu.Lock()
		delete(w.peers, whisperPeer)
		w.peerMu.Unlock()
	}()

	// Run the peer handshake and state updates
	if err := whisperPeer.handshake(); err != nil {
		return err
	}
	whisperPeer.start()
	defer whisperPeer.stop()

	// Read and process inbound messages directly to merge into client-global state
	for {
		// Fetch the next packet and decode the contained envelopes
		packet, err := rw.ReadMsg()
		if err != nil {
			return err
		}
		var envelopes []*Envelope
		if err := packet.Decode(&envelopes); err != nil {
			glog.V(logger.Info).Infof("%v: failed to decode envelope: %v", peer, err)
			continue
		}
		// Inject all envelopes into the internal pool
		for _, envelope := range envelopes {
			if err := w.add(envelope); err != nil {
				// TODO Punish peer here. Invalid envelope.
				glog.V(logger.Debug).Infof("%v: failed to pool envelope: %v", peer, err)
			}
			whisperPeer.mark(envelope)
		}
	}
}

// add inserts a new envelope into the message pool to be distributed within the
// whisper network. It also inserts the envelope into the expiration pool at the
// appropriate time-stamp.
func (w *Whisper) add(envelope *Envelope) error {
	w.poolMu.Lock()
	defer w.poolMu.Unlock()

	// Insert the message into the tracked pool
	hash := envelope.Hash()
	if _, ok := w.messages[hash]; ok {
		glog.V(logger.Detail).Infof("whisper envelope already cached: %x\n", envelope)
		return nil
	}
	w.messages[hash] = envelope

	// Insert the message into the expiration pool for later removal
	if w.expirations[envelope.Expiry] == nil {
		w.expirations[envelope.Expiry] = set.New(set.NonThreadSafe).(*set.SetNonTS)
	}
	if !w.expirations[envelope.Expiry].Has(hash) {
		w.expirations[envelope.Expiry].Add(hash)

		// Notify the local node of a message arrival
		go w.postEvent(envelope)
	}
	glog.V(logger.Detail).Infof("cached whisper envelope %x\n", envelope)

	return nil
}

// postEvent opens an envelope with the configured identities and delivers the
// message upstream from application processing.
func (w *Whisper) postEvent(envelope *Envelope) {
	if message := w.open(envelope); message != nil {
		w.filters.Notify(createFilter(message, envelope.Topics), message)
	}
}

// open tries to decrypt a whisper envelope with all the configured identities,
// returning the decrypted message and the key used to achieve it. If not keys
// are configured, open will return the payload as if non encrypted.
func (w *Whisper) open(envelope *Envelope) *Message {
	// Short circuit if no identity is set, and assume clear-text
	if len(w.keys) == 0 {
		if message, err := envelope.Open(nil); err == nil {
			return message
		}
	}
	// Iterate over the keys and try to decrypt the message
	for _, key := range w.keys {
		message, err := envelope.Open(key)
		if err == nil {
			message.To = &key.PublicKey
			return message
		} else if err == ecies.ErrInvalidPublicKey {
			return message
		}
	}
	// Failed to decrypt, don't return anything
	return nil
}

// createFilter creates a message filter to check against installed handlers.
func createFilter(message *Message, topics []Topic) filter.Filter {
	matcher := make([][]Topic, len(topics))
	for i, topic := range topics {
		matcher[i] = []Topic{topic}
	}
	return filterer{
		to:      string(crypto.FromECDSAPub(message.To)),
		from:    string(crypto.FromECDSAPub(message.Recover())),
		matcher: newTopicMatcher(matcher...),
	}
}

// update loops until the lifetime of the whisper node, updating its internal
// state by expiring stale messages from the pool.
func (w *Whisper) update() {
	// Start a ticker to check for expirations
	expire := time.NewTicker(expirationCycle)

	// Repeat updates until termination is requested
	for {
		select {
		case <-expire.C:
			w.expire()

		case <-w.quit:
			return
		}
	}
}

// expire iterates over all the expiration timestamps, removing all stale
// messages from the pools.
func (w *Whisper) expire() {
	w.poolMu.Lock()
	defer w.poolMu.Unlock()

	now := uint32(time.Now().Unix())
	for then, hashSet := range w.expirations {
		// Short circuit if a future time
		if then > now {
			continue
		}
		// Dump all expired messages and remove timestamp
		hashSet.Each(func(v interface{}) bool {
			delete(w.messages, v.(common.Hash))
			return true
		})
		w.expirations[then].Clear()
	}
}

// envelopes retrieves all the messages currently pooled by the node.
func (w *Whisper) envelopes() []*Envelope {
	w.poolMu.RLock()
	defer w.poolMu.RUnlock()

	envelopes := make([]*Envelope, 0, len(w.messages))
	for _, envelope := range w.messages {
		envelopes = append(envelopes, envelope)
	}
	return envelopes
}
