package transaction

import (
	"bytes"
	"encoding/json"
	"testing"
	// "time" // Unused
	// "shlyuz/pkg/component" // Unused
	"shlyuz/pkg/config" 
	"shlyuz/pkg/crypto" 
	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/instructions"
)

type TestComponent struct {
	ID               string
	InitialKeypair   asymmetric.AsymmetricKeyPair
	CurrentKeypair   asymmetric.AsymmetricKeyPair // Represents keypair whose PubKey is currently advertised for receiving.
	PeerPubKey       asymmetric.PublicKey         // The PubKey this component will encrypt to (peer's current receiving key).
	Config           config.ShlyuzConfig
}

func NewTestComponent(id string, xorKey int, initSig []byte) (*TestComponent, error) {
	initialKp, err := asymmetric.GenerateKeypair()
	if err != nil {
		return nil, err
	}
	return &TestComponent{
		ID:               id,
		InitialKeypair:   initialKp,
		CurrentKeypair:   initialKp, // Initially, current receiving key is the initial key.
		Config: config.ShlyuzConfig{
			Id: id,
			CryptoConfig: config.ShlyuzCrypto{ XorKey: xorKey, },
			InitSignature: initSig,
		},
	}, nil
}

func TestFullExchangeWithKeyRotation(t *testing.T) {
	compA, err := NewTestComponent("compA", 0x41, []byte("initSigA"))
	if err != nil { t.Fatalf("Failed to create compA: %v", err) }
	compB, err := NewTestComponent("compB", 0x42, []byte("initSigB"))
	if err != nil { t.Fatalf("Failed to create compB: %v", err) }

	t.Logf("CompA Initial/Current PubKey: %x", compA.CurrentKeypair.PubKey[:6])
	t.Logf("CompB Initial/Current PubKey: %x", compB.CurrentKeypair.PubKey[:6])

	// --- Initial Handshake ---
	// CompA sends 'ii' (sealed) to CompB
	initFrameAArgs := instructions.Transaction{ Cmd: "ii", ComponentId: compA.ID }
	initFrameA := instructions.CreateInstructionFrame(initFrameAArgs, true)
	initFrameA.Pk = compA.CurrentKeypair.PubKey // A's current (initial) PubKey
	marshaledInitFrameA, _ := json.Marshal(initFrameA)
	sealedMsgA := crypto.PrepareSealedFrame(marshaledInitFrameA, compB.InitialKeypair.PubKey, compA.Config.CryptoConfig.XorKey, compA.Config.InitSignature)
	
	unwrappedInitA := crypto.UnwrapSealedFrame(sealedMsgA, compB.InitialKeypair.PrivKey, compB.InitialKeypair.PubKey, compA.Config.CryptoConfig.XorKey, compA.Config.InitSignature)
	if unwrappedInitA == nil { t.Fatalf("CompB failed to unwrap 'ii'") }
	var receivedInitFrameA instructions.InstructionFrame
	json.Unmarshal(unwrappedInitA, &receivedInitFrameA)
	compB.PeerPubKey = receivedInitFrameA.Pk // B learns A's current (initial) PubKey
	t.Logf("CompB learned CompA's PubKey: %x", compB.PeerPubKey[:6])

	// CompB sends 'ipi' (transmit) to CompA
	compB_signing_keypair_ipi := compB.CurrentKeypair // B's initial keypair (CurrentKeypair is still InitialKeypair)
	keypairB_next_for_receiving_after_ipi, _ := asymmetric.GenerateKeypair() // B will advertise this.

	initAckFrameBArgs := instructions.Transaction{ Cmd: "ipi", ComponentId: compB.ID }
	initAckFrameB := instructions.CreateInstructionFrame(initAckFrameBArgs, true)
	initAckFrameB.Pk = keypairB_next_for_receiving_after_ipi.PubKey // B advertises its *next* receiving key.
	marshaledInitAckB, _ := json.Marshal(initAckFrameB)
	
	transmitMsgB_ipi := crypto.PrepareTransmitFrame(marshaledInitAckB, compB.PeerPubKey /*A's initial PubKey*/, compB_signing_keypair_ipi.PrivKey, compB.Config.CryptoConfig.XorKey)
	
	// CompA receives 'ipi'
	// A's CurrentKeypair is still its InitialKeypair.
	unwrappedAckB := crypto.UnwrapTransmitFrame(transmitMsgB_ipi, compB_signing_keypair_ipi.PubKey, compA.CurrentKeypair.PrivKey, compB.Config.CryptoConfig.XorKey)
	if unwrappedAckB == nil { t.Fatalf("CompA failed to unwrap 'ipi'") }
	var receivedAckFrameB instructions.InstructionFrame
	json.Unmarshal(unwrappedAckB, &receivedAckFrameB)
	compA.PeerPubKey = receivedAckFrameB.Pk // A learns B's *next* receiving key.
	t.Logf("CompA learned CompB's next PubKey: %x", compA.PeerPubKey[:6])

	// CompB now rotates its CurrentKeypair to the one it advertised in 'ipi'.
	// This key will be used by B to receive A's first 'icmdr'.
	compB.CurrentKeypair = keypairB_next_for_receiving_after_ipi
	t.Logf("CompB (sender of ipi) rotated its CurrentKeypair. New PubKey for B (for receiving): %x", compB.CurrentKeypair.PubKey[:6])

	// --- Request-Response Cycles ---
	for i := 0; i < 3; i++ {
		t.Logf("\n--- Cycle %d ---", i+1)

		// CompA: Send 'icmdr'
		compA_signing_keypair := compA.CurrentKeypair // A uses this (A_kp_N) to sign.
		keypairA_next_for_receiving, _ := asymmetric.GenerateKeypair() // A will advertise this (A_kp_N+1) for receiving next.
		
		cmdArgs := "txA_cycle" + string(rune(i+'0')) // Create unique TxId per cycle for clarity
		cmdFrameAArgs := instructions.Transaction{ Cmd: "icmdr", TxId: cmdArgs, ComponentId: compA.ID }
		cmdFrameA := instructions.CreateInstructionFrame(cmdFrameAArgs, true)
		cmdFrameA.Pk = keypairA_next_for_receiving.PubKey // A advertises A_kp_N+1.PubKey.
		marshaledCmdA, _ := json.Marshal(cmdFrameA)

		t.Logf("CompA [Send icmdr %d]: Advertised Pk (A's next recv key): %x", i+1, cmdFrameA.Pk[:6])
		t.Logf("CompA [Send icmdr %d]: Encrypts to B's current recv PubKey (compA.PeerPubKey): %x", i+1, compA.PeerPubKey[:6])
		t.Logf("CompA [Send icmdr %d]: Signs with its current PrivKey (of PubKey %x)", i+1, compA_signing_keypair.PubKey[:6])
		transmitMsgA := crypto.PrepareTransmitFrame(marshaledCmdA, compA.PeerPubKey, compA_signing_keypair.PrivKey, compA.Config.CryptoConfig.XorKey)
		
		// CompB: Receive 'icmdr'
		// Message was encrypted for B's CurrentKeypair.PubKey (which is compA.PeerPubKey).
		// B uses its CurrentKeypair.PrivKey for decryption.
		// Sender's signing key (for verification) is compA_signing_keypair.PubKey.
		t.Logf("CompB [Recv icmdr %d]: Expects sender's signing PubKey: %x", i+1, compA_signing_keypair.PubKey[:6])
		t.Logf("CompB [Recv icmdr %d]: Uses its CurrentKeypair.PrivKey (for PubKey %x) for decryption", i+1, compB.CurrentKeypair.PubKey[:6])
		
		unwrappedCmdA := crypto.UnwrapTransmitFrame(transmitMsgA, compA_signing_keypair.PubKey, compB.CurrentKeypair.PrivKey, compA.Config.CryptoConfig.XorKey)
		if unwrappedCmdA == nil { t.Fatalf("Cycle %d: CompB failed to unwrap 'icmdr'", i+1) }
		var receivedCmdFrameA instructions.InstructionFrame
		json.Unmarshal(unwrappedCmdA, &receivedCmdFrameA)
		
		if !bytes.Equal(receivedCmdFrameA.Pk[:], keypairA_next_for_receiving.PubKey[:]) {
			t.Fatalf("Cycle %d: Pk in received icmdr (A's next key) mismatch. Got %x, want %x", i+1, receivedCmdFrameA.Pk[:6], keypairA_next_for_receiving.PubKey[:6])
		}
		// B learns A's *next* receiving key. This is what B will target in its reply.
		compB.PeerPubKey = receivedCmdFrameA.Pk 
		t.Logf("CompB [Recv icmdr %d]: Updated PeerPubKey for A to: %x", i+1, compB.PeerPubKey[:6])

		// CompA now ACTUALLY rotates its CurrentKeypair to the one it advertised.
		// This key (keypairA_next_for_receiving) will be used by A to decrypt B's response in this cycle.
		compA.CurrentKeypair = keypairA_next_for_receiving
		t.Logf("CompA [Post Recv by B %d]: Rotated its CurrentKeypair (for receiving). New PubKey: %x", i+1, compA.CurrentKeypair.PubKey[:6])

		// CompB: Send 'fcmd'
		compB_signing_keypair := compB.CurrentKeypair // B uses this (B_kp_N) to sign.
		keypairB_next_for_receiving, _ := asymmetric.GenerateKeypair() // B will advertise this (B_kp_N+1) for receiving next.

		respArgsStr := `{"output":"cycle ` + string(rune(i+'1')) + ` done"}`
		respFrameBArgs := instructions.Transaction{ Cmd: "fcmd", TxId: receivedCmdFrameA.TxId, ComponentId: compB.ID, Arg: []byte(respArgsStr) }
		respFrameB := instructions.CreateInstructionFrame(respFrameBArgs, true)
		respFrameB.Pk = keypairB_next_for_receiving.PubKey // B advertises B_kp_N+1.PubKey.
		marshaledRespB, _ := json.Marshal(respFrameB)

		t.Logf("CompB [Send fcmd %d]: Advertised Pk (B's next recv key): %x", i+1, respFrameB.Pk[:6])
		// CompB encrypts to A's *current* receiving key (compB.PeerPubKey, which B just learned from A's icmdr).
		t.Logf("CompB [Send fcmd %d]: Encrypts to A's current recv PubKey (compB.PeerPubKey): %x", i+1, compB.PeerPubKey[:6])
		t.Logf("CompB [Send fcmd %d]: Signs with its current PrivKey (of PubKey %x)", i+1, compB_signing_keypair.PubKey[:6])
		transmitMsgBResp := crypto.PrepareTransmitFrame(marshaledRespB, compB.PeerPubKey, compB_signing_keypair.PrivKey, compB.Config.CryptoConfig.XorKey)
		
		// CompA: Receive 'fcmd'
		// Message was encrypted for A's *current* receiving key (compA.CurrentKeypair).
		// Sender's signing key (for verification) is compB_signing_keypair.PubKey.
		t.Logf("CompA [Recv fcmd %d]: Expects sender's signing PubKey: %x", i+1, compB_signing_keypair.PubKey[:6])
		t.Logf("CompA [Recv fcmd %d]: Uses its CurrentKeypair.PrivKey (for PubKey %x) for decryption", i+1, compA.CurrentKeypair.PubKey[:6])
		
		unwrappedRespB := crypto.UnwrapTransmitFrame(transmitMsgBResp, compB_signing_keypair.PubKey, compA.CurrentKeypair.PrivKey, compB.Config.CryptoConfig.XorKey)
		if unwrappedRespB == nil { t.Fatalf("Cycle %d: CompA failed to unwrap 'fcmd'", i+1) }
		var receivedRespFrameB instructions.InstructionFrame
		json.Unmarshal(unwrappedRespB, &receivedRespFrameB)
		if !bytes.Equal(receivedRespFrameB.Pk[:], keypairB_next_for_receiving.PubKey[:]) {
			t.Fatalf("Cycle %d: Pk in received fcmd (B's next key) mismatch. Got %x, want %x", i+1, receivedRespFrameB.Pk[:6], keypairB_next_for_receiving.PubKey[:6])
		}
		compA.PeerPubKey = receivedRespFrameB.Pk // A learns B's *next* receiving key.
		t.Logf("CompA [Recv fcmd %d]: Updated PeerPubKey for B to: %x", i+1, compA.PeerPubKey[:6])

		// CompB rotates its CurrentKeypair to what it advertised.
		compB.CurrentKeypair = keypairB_next_for_receiving
		t.Logf("CompB [Post Recv by A %d]: Rotated its CurrentKeypair. New PubKey: %x", i+1, compB.CurrentKeypair.PubKey[:6])

		// Assert contents
		if receivedCmdFrameA.Cmd != "icmdr" { t.Errorf("Cycle %d: Expected Cmd 'icmdr'", i+1) }
		if receivedRespFrameB.Cmd != "fcmd" { t.Errorf("Cycle %d: Expected Cmd 'fcmd'", i+1) }
		if receivedRespFrameB.CmdArgs != respArgsStr { t.Errorf("Cycle %d: CmdArgs mismatch. Got '%s', want '%s'", i+1, receivedRespFrameB.CmdArgs, respArgsStr) }
		t.Logf("Cycle %d completed successfully.", i+1)
	}
	t.Log("TestFullExchangeWithKeyRotation completed.")
}
