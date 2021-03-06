package responding_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

// Test the functionality of the CandidateBroker.
func TestCandidateBroker(t *testing.T) {
	rb := rpcbus.New()
	respChan := make(chan *bytes.Buffer, 1)
	c := responding.NewCandidateBroker(rb, respChan)
	blockHash, _ := crypto.RandEntropy(32)
	quitChan := provideCandidate(rb, blockHash)

	// First, ask for the wrong candidate.
	wrongHash, _ := crypto.RandEntropy(32)
	assert.Error(t, c.ProvideCandidate(bytes.NewBuffer(wrongHash)))

	// Now, ask for the correct one.
	assert.NoError(t, c.ProvideCandidate(bytes.NewBuffer(blockHash)))

	// Should receive something on `respChan`
	<-respChan

	// Clean up goroutine
	quitChan <- struct{}{}
}

func provideCandidate(rb *rpcbus.RPCBus, correctHash []byte) chan struct{} {
	quitChan := make(chan struct{}, 1)
	reqChan := make(chan rpcbus.Request, 1)

	rb.Register(topics.GetCandidate, reqChan)

	go func(reqChan chan rpcbus.Request, quitChan chan struct{}, correctHash []byte) {
		for {
			select {
			case r := <-reqChan:
				params := r.Params.(bytes.Buffer)
				hash := make([]byte, 32)
				if err := encoding.Read256(&params, hash); err != nil {
					panic(err)
				}

				if bytes.Equal(hash, correctHash) {
					blk := helper.RandomBlock(&testing.T{}, 2, 1)
					cm := message.NewCandidate()
					cm.Block = blk
					r.RespChan <- rpcbus.Response{*cm, nil}
					continue
				}

				r.RespChan <- rpcbus.Response{nil, errors.New("not found")}
			case <-quitChan:
				return
			}
		}
	}(reqChan, quitChan, correctHash)

	return quitChan
}
