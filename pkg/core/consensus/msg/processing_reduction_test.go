package msg_test

import (
	"encoding/hex"
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

func TestVerifyReduction(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	// Add the block to our collection
	hashStr := hex.EncodeToString(emptyBlock.Header.Hash)
	ctx.CandidateBlocks[hashStr] = emptyBlock

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Verify the message
	err2 := msg.Process(ctx, m)
	if err2 != nil {
		t.Fatal(err2)
	}
}

func TestReductionUnknownBlock(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Verify the message (should fail with low priority error)
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("unknown block check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestReductionWrongStep(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add the block to our collection
	hashStr := hex.EncodeToString(emptyBlock.Header.Hash)
	ctx.CandidateBlocks[hashStr] = emptyBlock

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Change our step and verify the message (should fail with low priority error)
	ctx.Step++
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("step check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestReductionNotInCommittee(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add the block to our collection
	hashStr := hex.EncodeToString(emptyBlock.Header.Hash)
	ctx.CandidateBlocks[hashStr] = emptyBlock

	// Verify the message (should fail with low priority error)
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("step check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestReductionWrongBLSKey(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add the block to our collection
	hashStr := hex.EncodeToString(emptyBlock.Header.Hash)
	ctx.CandidateBlocks[hashStr] = emptyBlock

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Clear out our bls key mapping
	ctx.NodeBLS = make(map[string][]byte)

	// Verify the message (should fail with low priority error)
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("step check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestReductionWrongBLSSig(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, true)
	if err != nil {
		t.Fatal(err)
	}

	// Add the block to our collection
	hashStr := hex.EncodeToString(emptyBlock.Header.Hash)
	ctx.CandidateBlocks[hashStr] = emptyBlock

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Verify the message (should fail with low priority error)
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("step check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}
