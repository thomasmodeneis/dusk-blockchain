package reduction

import (
	"bytes"
	"errors"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// LaunchSigSetReducer creates and wires a broker, initiating the components that
// have to do with Signature Set Reduction
func LaunchSigSetReducer(eventBus *wire.EventBus, committee committee.Committee,
	timeout time.Duration) *broker {

	sigSetChan := selection.InitBestSigSetUpdate(eventBus)
	handler := newSigSetHandler(eventBus, committee)
	broker := newBroker(eventBus, handler, sigSetChan, committee,
		topics.SigSetReduction,
		string(msg.OutgoingSigSetReductionTopic),
		string(msg.OutgoingSigSetAgreementTopic),
		string(msg.SigSetGenerationTopic), timeout)
	go broker.Listen()
	return broker
}

type (
	// SigSetCollector is the public collector used outside of the Broker (which use the unexported one)
	SigSetCollector struct{}

	// SigSetEvent is the event related to the completed reduction of a Signature Set for a specific round
	SigSetEvent struct {
		*committee.ReductionEvent
		BlockHash []byte
	}

	SigSetUnmarshaller struct {
		*unMarshaller
	}

	// sigSetHandler is responsible for performing operations that need to know
	// about specific event fields.
	sigSetHandler struct {
		committee committee.Committee
		*SigSetUnmarshaller
		blockHash []byte
	}
)

// Equal implements Event interface.
func (sse *SigSetEvent) Equal(e wire.Event) bool {
	return sse.ReductionEvent.Equal(e) &&
		bytes.Equal(sse.BlockHash, e.(*SigSetEvent).BlockHash)
}

func NewSigSetUnMarshaller(validate func([]byte, []byte, []byte) error) *SigSetUnmarshaller {
	return &SigSetUnmarshaller{
		unMarshaller: newUnMarshaller(validate),
	}
}

func (ssru *SigSetUnmarshaller) Unmarshal(r *bytes.Buffer, e wire.Event) error {
	sigSetEvent := e.(*SigSetEvent)
	if err := ssru.unMarshaller.Unmarshal(r, sigSetEvent.ReductionEvent); err != nil {
		return err
	}

	if err := encoding.Read256(r, &sigSetEvent.BlockHash); err != nil {
		return err
	}

	return nil
}

func (ssru *SigSetUnmarshaller) Marshal(r *bytes.Buffer, e wire.Event) error {
	sigSetEvent := e.(*SigSetEvent)
	if err := ssru.EventHeaderMarshaller.Marshal(r, sigSetEvent.EventHeader); err != nil {
		return err
	}

	if err := ssru.unMarshaller.Marshal(r, sigSetEvent.ReductionEvent); err != nil {
		return err
	}

	if err := encoding.Write256(r, sigSetEvent.BlockHash); err != nil {
		return err
	}

	return nil
}

func (ssru *SigSetUnmarshaller) UnmarshalVoteSet(r *bytes.Buffer) ([]wire.Event, error) {
	length, err := encoding.ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	evs := make([]wire.Event, length)
	for i := uint64(0); i < length; i++ {
		rev := &SigSetEvent{
			ReductionEvent: &committee.ReductionEvent{
				EventHeader: &consensus.EventHeader{},
			},
		}
		if err := ssru.Unmarshal(r, rev); err != nil {
			return nil, err
		}

		evs[i] = rev
	}

	return evs, nil
}

func (ssru *SigSetUnmarshaller) MarshalVoteSet(r *bytes.Buffer, evs []wire.Event) error {
	if err := encoding.WriteVarInt(r, uint64(len(evs))); err != nil {
		return err
	}

	for _, event := range evs {
		if err := ssru.Marshal(r, event); err != nil {
			return err
		}
	}

	return nil
}

// newSigSetHandler will return a SigSetHandler, injected with the passed committee and an unmarshaller
func newSigSetHandler(eventBus *wire.EventBus, committee committee.Committee) *sigSetHandler {
	phaseChannel := consensus.InitPhaseUpdate(eventBus)
	sigSetHandler := &sigSetHandler{
		committee:          committee,
		SigSetUnmarshaller: NewSigSetUnMarshaller(msg.VerifyEd25519Signature),
	}

	go func() {
		for {
			sigSetHandler.blockHash = <-phaseChannel
		}
	}()
	return sigSetHandler
}

// Priority is not used for this handler
func (s *sigSetHandler) Priority(ev1, ev2 wire.Event) wire.Event {
	return nil
}

// NewEvent returns a sigSetEvent
func (s *sigSetHandler) NewEvent() wire.Event {
	return &SigSetEvent{
		ReductionEvent: &committee.ReductionEvent{
			EventHeader: &consensus.EventHeader{},
		},
	}
}

func (s *sigSetHandler) ExtractHeader(e wire.Event, h *consensus.EventHeader) {
	ev := e.(*committee.ReductionEvent)
	h.Round = ev.Round
	h.Step = ev.Step
}

func (s *sigSetHandler) EmbedVoteHash(e wire.Event, r *bytes.Buffer) error {
	var votedHash, blockHash []byte
	if e == nil {
		votedHash, blockHash = make([]byte, 32), make([]byte, 32)
	} else {
		ev := e.(*SigSetEvent)
		votedHash, blockHash = ev.VotedHash, ev.BlockHash
	}
	if err := encoding.Write256(r, votedHash); err != nil {
		return err
	}
	if err := encoding.Write256(r, blockHash); err != nil {
		return err
	}
	return nil
}

// Verify the sigSetEvent
func (s *sigSetHandler) Verify(e wire.Event) error {
	ev := e.(*SigSetEvent)
	if err := msg.VerifyBLSSignature(ev.PubKeyBLS, ev.VotedHash, ev.SignedHash); err != nil {
		return err
	}

	if !bytes.Equal(s.blockHash, ev.BlockHash) {
		return errors.New("sig set handler: block hash mismatch")
	}

	if !s.committee.IsMember(ev.PubKeyBLS) {
		return errors.New("sig set handler: voter not eligible to vote")
	}

	return nil
}