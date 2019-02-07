package payload_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

func TestMsgCertificateReqEncodeDecode(t *testing.T) {
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	msg, err := payload.NewMsgCertificateReq(1500, hash)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &payload.MsgCertificateReq{}
	msg2.Decode(buf)

	assert.Equal(t, msg, msg2)
}

// Check to see whether length checks are working.
func TestMsgCertificateReqChecks(t *testing.T) {
	wrongHash, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := payload.NewMsgCertificateReq(200, wrongHash); err == nil {
		t.Fatal("check for hash did not work")
	}
}
