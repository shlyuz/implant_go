package crypto_test

import (
	"bytes"
	"encoding/json"
	"log" // Keep one log import
	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/instructions"
	"shlyuz/pkg/utils/logging"
	"testing"

	// Import the package being tested
	"shlyuz/pkg/crypto"
)

// Helper to create a sample InstructionFrame and marshal it
func createSampleDataFrame(t *testing.T, componentID, cmd, cmdArgs, date, txID string, pk *asymmetric.PublicKey) []byte {
	sampleInstruction := instructions.InstructionFrame{
		ComponentId: componentID,
		Cmd:         cmd,
		CmdArgs:     cmdArgs,
		Date:        date,
		TxId:        txID,
	}
	if pk != nil {
		sampleInstruction.Pk = *pk // Dereference if pk is *asymmetric.PublicKey
	}
	dataFrame, err := json.Marshal(sampleInstruction)
	if err != nil {
		t.Fatalf("Failed to marshal sampleInstruction: %v", err)
	}
	return dataFrame
}

func TestPrepareAndUnwrapSealedFrame(t *testing.T) {
	log.SetPrefix(logging.GetLogPrefix())
	t.Log("Starting TestPrepareAndUnwrapSealedFrame")

	dataFrame := createSampleDataFrame(t, "testSealedComp", "testSealedCmd", "{\"argS\":\"valS\"}", "2023-10-26T10:00:00Z", "txSealed123", nil)

	myKeyPair, err := asymmetric.GenerateKeypair()
	if err != nil {
		t.Fatalf("Failed to generate myKeyPair: %v", err)
	}
	t.Logf("MyKeyPair for Sealed: PubKey = %x", myKeyPair.PubKey[:])

	xorKey := 0x42
	initSig := []byte("testsignature_sealed") // Length 20
	t.Logf("Original initSig: %x (len %d)", initSig, len(initSig))

	// PrepareSealedFrame now returns only []byte
	sealedMsg := crypto.PrepareSealedFrame(dataFrame, myKeyPair.PubKey, xorKey, initSig)
	if len(sealedMsg) == 0 {
		t.Fatalf("PrepareSealedFrame returned an empty message")
	}
	// impKeyPairSender is no longer returned, so logging its PubKey is removed.
	t.Logf("SealedMsg length: %d.", len(sealedMsg))
	if len(sealedMsg) < len(initSig) {
		t.Fatalf("Sealed message is shorter than initSig. Length: %d, initSig Length: %d", len(sealedMsg), len(initSig))
	}
	t.Logf("SealedMsg prefix (expected initSig part): %x", sealedMsg[:len(initSig)])


	t.Log("Attempting UnwrapSealedFrame (correct case)...")
	unwrappedDataFrame := crypto.UnwrapSealedFrame(sealedMsg, myKeyPair.PrivKey, myKeyPair.PubKey, xorKey, initSig)
	if unwrappedDataFrame == nil {
		t.Errorf("UnwrapSealedFrame (correct case) returned nil")
	} else if !bytes.Equal(unwrappedDataFrame, dataFrame) {
		t.Errorf("UnwrapSealedFrame (correct case) data mismatch: got %s, want %s", string(unwrappedDataFrame), string(dataFrame))
	} else {
		t.Log("UnwrapSealedFrame (correct case) successful.")
	}

	initSigBad := []byte("bad_sealed_signature!!") // Ensure different and comparable length
	t.Logf("Attempting UnwrapSealedFrame with bad initSig: %x (len %d)", initSigBad, len(initSigBad))
	unwrappedDataFrameBadSig := crypto.UnwrapSealedFrame(sealedMsg, myKeyPair.PrivKey, myKeyPair.PubKey, xorKey, initSigBad)
	if unwrappedDataFrameBadSig != nil {
		t.Errorf("UnwrapSealedFrame with mismatched initSig should have returned nil, but got data: %s", string(unwrappedDataFrameBadSig))
	} else {
		t.Log("UnwrapSealedFrame with mismatched initSig correctly returned nil.")
	}

	wrongKeyPair, _ := asymmetric.GenerateKeypair()
	t.Logf("Attempting UnwrapSealedFrame with wrong private key (Correct PubKey: %x, Wrong PrivKey corresponding to PubKey: %x)", myKeyPair.PubKey[:], wrongKeyPair.PubKey[:])
	unwrappedDataFrameWrongPrivK := crypto.UnwrapSealedFrame(sealedMsg, wrongKeyPair.PrivKey, myKeyPair.PubKey, xorKey, initSig)
	if unwrappedDataFrameWrongPrivK != nil {
		t.Errorf("UnwrapSealedFrame with wrong private key should have returned nil, got: %s", string(unwrappedDataFrameWrongPrivK))
	} else {
		t.Log("UnwrapSealedFrame with wrong private key correctly returned nil.")
	}
	t.Log("Finished TestPrepareAndUnwrapSealedFrame")
}

func TestPrepareAndUnwrapTransmitFrame(t *testing.T) {
	log.SetPrefix(logging.GetLogPrefix())
	t.Log("Starting TestPrepareAndUnwrapTransmitFrame")

	senderKeyPair, err := asymmetric.GenerateKeypair()
	if err != nil {
		t.Fatalf("Failed to generate senderKeyPair: %v", err)
	}
	receiverKeyPair, err := asymmetric.GenerateKeypair()
	if err != nil {
		t.Fatalf("Failed to generate receiverKeyPair: %v", err)
	}
	t.Logf("Sender Initial: PubKey = %x", senderKeyPair.PubKey[:])
	t.Logf("Receiver Initial: PubKey = %x", receiverKeyPair.PubKey[:])

	// This Pk is what the sender (current code execution path) wants the receiver to use for the *receiver's next message to the sender*.
	// For this test, let's say the sender wants the receiver to use senderKeyPair.PubKey.
	pkForInstruction := senderKeyPair.PubKey 
	dataFrame := createSampleDataFrame(t, "testTransmitComp", "testTransmitCmd", "{\"argT\":\"valT\"}", "2023-10-26T11:00:00Z", "txTransmit456", &pkForInstruction)
	t.Logf("DataFrame created with Pk field: %x", pkForInstruction[:])

	xorKey := 0x43

	// Sender Action:
	// Encrypts dataFrame (which contains pkForInstruction) using receiverKeyPair.PubKey.
	// Signs/Auths using senderKeyPair.PrivKey.
	// Returns senderNextUsageKeyPair, which the sender will use for *its* next message.
	t.Logf("PrepareTransmitFrame: encrypt To = %x, sign From = %x", receiverKeyPair.PubKey[:], senderKeyPair.PrivKey[:])
	// PrepareTransmitFrame now returns only []byte
	transmitMsg := crypto.PrepareTransmitFrame(dataFrame, receiverKeyPair.PubKey, senderKeyPair.PrivKey, xorKey)
	if len(transmitMsg) == 0 {
		t.Fatalf("PrepareTransmitFrame returned an empty message")
	}
	// senderNextUsageKeyPair is no longer returned. The key rotation logic is now handled in the transaction layer.
	t.Logf("PrepareTransmitFrame successful.")

	// Receiver Action:
	// Decrypts using receiverKeyPair.PrivKey.
	// Verifies signature/Auth using senderKeyPair.PubKey (as peerPubKey).
	t.Logf("UnwrapTransmitFrame: decrypt With = %x, verify From = %x", receiverKeyPair.PrivKey[:], senderKeyPair.PubKey[:])
	unwrappedDataFrame := crypto.UnwrapTransmitFrame(transmitMsg, senderKeyPair.PubKey, receiverKeyPair.PrivKey, xorKey)
	if unwrappedDataFrame == nil {
		t.Fatalf("UnwrapTransmitFrame (correct case) returned nil")
	}
	if !bytes.Equal(unwrappedDataFrame, dataFrame) {
		t.Errorf("UnwrapTransmitFrame (correct case) data mismatch: got %s, want %s", string(unwrappedDataFrame), string(dataFrame))
		var gotI, wantI instructions.InstructionFrame
		json.Unmarshal(unwrappedDataFrame, &gotI)
		json.Unmarshal(dataFrame, &wantI)
		t.Logf("Got Unmarshaled: %+v", gotI)
		t.Logf("Want Unmarshaled: %+v", wantI)
	} else {
		t.Log("UnwrapTransmitFrame (correct case) successful.")
	}

	var unwrappedInstruction instructions.InstructionFrame
	err = json.Unmarshal(unwrappedDataFrame, &unwrappedInstruction)
	if err != nil {
		t.Fatalf("Failed to unmarshal unwrappedDataFrame: %v. Content: %s", err, string(unwrappedDataFrame))
	}
	// The Pk in the instruction is the one the sender wants the receiver to use for the *receiver's* next message to the *sender*.
	if !bytes.Equal(unwrappedInstruction.Pk[:], pkForInstruction[:]) {
		t.Errorf("Key rotation check failed: Unwrapped instruction's Pk (%x) does not match sender's intended next public key for receiver (%x)", unwrappedInstruction.Pk[:], pkForInstruction[:])
	} else {
		t.Logf("Key rotation Pk check successful. Unwrapped Pk: %x", unwrappedInstruction.Pk[:])
	}

	wrongXorKey := 0x99
	t.Logf("Attempting UnwrapTransmitFrame with wrong XOR key (0x%x)", wrongXorKey)
	unwrappedDataFrameWrongXor := crypto.UnwrapTransmitFrame(transmitMsg, senderKeyPair.PubKey, receiverKeyPair.PrivKey, wrongXorKey)
	if unwrappedDataFrameWrongXor != nil && bytes.Equal(unwrappedDataFrameWrongXor, dataFrame) {
		t.Errorf("UnwrapTransmitFrame with wrong XOR key should have failed or returned different data, but it matched.")
	} else {
		t.Log("UnwrapTransmitFrame with wrong XOR key behaved as expected (nil or different data).")
	}

	anotherKeyPair, _ := asymmetric.GenerateKeypair()
	t.Logf("Attempting UnwrapTransmitFrame with wrong peer (sender's) public key (%x)", anotherKeyPair.PubKey[:])
	unwrappedDataFrameWrongPeerPub := crypto.UnwrapTransmitFrame(transmitMsg, anotherKeyPair.PubKey, receiverKeyPair.PrivKey, xorKey)
	if unwrappedDataFrameWrongPeerPub != nil && bytes.Equal(unwrappedDataFrameWrongPeerPub, dataFrame) {
		t.Errorf("UnwrapTransmitFrame with wrong peer public key should have failed or returned different data")
	} else {
		t.Log("UnwrapTransmitFrame with wrong peer public key behaved as expected (nil or different data).")
	}

	t.Logf("Attempting UnwrapTransmitFrame with wrong receiver private key (key for pub %x)", anotherKeyPair.PubKey[:])
	unwrappedDataFrameWrongMyPriv := crypto.UnwrapTransmitFrame(transmitMsg, senderKeyPair.PubKey, anotherKeyPair.PrivKey, xorKey)
	if unwrappedDataFrameWrongMyPriv != nil && bytes.Equal(unwrappedDataFrameWrongMyPriv, dataFrame) {
		t.Errorf("UnwrapTransmitFrame with wrong receiver private key should have failed or returned different data")
	} else {
		t.Log("UnwrapTransmitFrame with wrong receiver private key behaved as expected (nil or different data).")
	}
	t.Log("Finished TestPrepareAndUnwrapTransmitFrame")
}
