package block_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeHeader(t *testing.T) {

	assert := assert.New(t)

	// Create a random header
	hdr := helper.RandomHeader(t, 200)
	err := hdr.SetHash()
	assert.Nil(err)

	// Encode header into a buffer
	buf := new(bytes.Buffer)
	err = block.MarshalHeader(buf, hdr)
	assert.Nil(err)

	// Decode buffer into a header struct
	decHdr := block.NewHeader()
	err = block.UnmarshalHeader(buf, decHdr)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(hdr.Equals(decHdr))
}
