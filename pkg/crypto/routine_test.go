package routine

import (
	"bytes"
	"shlyuz/pkg/crypto/asymmetric"
	"testing"
)

var testcase = []struct {
	dataframe []byte
}{
	{
		[]byte(`{"component": "deadbeef", "cmd": "foobar", "args": [{"ayyy": "lmao"}]}`),
	},
}

type tinyRegisteredComponent struct {
	CurPubKey  asymmetric.PublicKey
	CurKeyPair asymmetric.AsymmetricKeyPair
	XorKey     int
}

func TestRoutineTransmitFrame(t *testing.T) {
	t.Parallel()
	var err error
	impKeyPair, err := asymmetric.GenerateKeypair()
	if err != nil {
		t.Fatal("[FAIL] unable to generate Imp keypair: ", err)
	}
	lpKeyPair, err := asymmetric.GenerateKeypair()
	if err != nil {
		t.Fatal("[FAIL] unable to generate lp keypair: ", err)
	}
	for _, test := range testcase {
		t.Log("Testing implant creation of transmit frame")
		encryptedTransmitFrame, _ := PrepareTransmitFrame(test.dataframe, lpKeyPair.PubKey, impKeyPair.PrivKey, 12)
		if bytes.Equal(test.dataframe, encryptedTransmitFrame) {
			t.Fatal("[FAIL] implant's PrepareTransmitFrame did not work properly")
		}
		t.Log("Testing lp's ability to unwrap LP's transmit frame")
		decryptedTransmitFrame := UnwrapTransmitFrame(encryptedTransmitFrame, impKeyPair.PubKey, lpKeyPair.PrivKey, 12)
		if bytes.Equal(encryptedTransmitFrame, decryptedTransmitFrame) {
			t.Fatal("[FAIL] UnwrapTransmitFrame did not work properly")
		}
		if !bytes.Equal(decryptedTransmitFrame, test.dataframe) {
			t.Fatal("[FAIL] UnwrapTransmitFrame did not produce expected result. Got: ", decryptedTransmitFrame, " Wanted: ", test.dataframe)
		}
	}
}

func TestKeyRotationRoutine(t *testing.T) {
	t.Parallel()
	var err error
	var testLpReg tinyRegisteredComponent
	testLpReg.CurKeyPair, err = asymmetric.GenerateKeypair()
	if err != nil {
		t.Fatal("[FAIL] unable to generate implant keypair for testLpReg: ", err)
	}
	lpKeyPair, err := asymmetric.GenerateKeypair()
	if err != nil {
		t.Fatal("[FAIL] unable to generate lp keypair used with testLpReg: ", err)
	}
	t.Log("Registering test LP in test Implant")
	testLpReg.CurPubKey = lpKeyPair.PubKey
	testLpReg.XorKey = 69
	for _, test := range testcase {
		var encryptedSrcFrame []byte
		var decryptedSrcFrame []byte
		t.Log("Generating Implant Transmit Message")
		encryptedSrcFrame, newImpKeyPair := PrepareTransmitFrame(test.dataframe, testLpReg.CurPubKey, testLpReg.CurKeyPair.PrivKey, testLpReg.XorKey)
		if bytes.Equal(test.dataframe, encryptedSrcFrame) {
			t.Fatal("[FAIL] implant's PrepareTransmitFrame did not work properly")
		}
		t.Log("LP Decrypted implant's generated test message")
		decryptedSrcFrame = UnwrapTransmitFrame(encryptedSrcFrame, testLpReg.CurKeyPair.PubKey, lpKeyPair.PrivKey, testLpReg.XorKey)
		if !bytes.Equal(test.dataframe, decryptedSrcFrame) {
			t.Fatal("[FAIL] LP's inital UnwrapTransmitFrame did not work properly")
		}
		t.Log("Rotating Updating Implant's keys")
		testLpReg.CurKeyPair = newImpKeyPair
		t.Log("Generating rotated Implant Transmit Message")
		encryptedSrcFrame, _ = PrepareTransmitFrame(test.dataframe, testLpReg.CurPubKey, testLpReg.CurKeyPair.PrivKey, testLpReg.XorKey)
		if bytes.Equal(test.dataframe, encryptedSrcFrame) {
			t.Fatal("[FAIL] implant's rotated PrepareTransmitFrame did not work properly")
		}
		t.Log("LP Decrypted implant's rotated generated test message")
		decryptedSrcFrame = UnwrapTransmitFrame(encryptedSrcFrame, testLpReg.CurKeyPair.PubKey, lpKeyPair.PrivKey, testLpReg.XorKey)
		if !bytes.Equal(test.dataframe, decryptedSrcFrame) {
			t.Fatal("[FAIL] LP's rotated UnwrapTransmitFrame did not work properly")
		}
		t.Log("Generating LP Transmit Message using new keys")
		encryptedSrcFrame, _ = PrepareTransmitFrame(test.dataframe, testLpReg.CurKeyPair.PubKey, lpKeyPair.PrivKey, testLpReg.XorKey)
		if bytes.Equal(test.dataframe, encryptedSrcFrame) {
			t.Fatal("[FAIL] LP's rotated PrepareTransmitFrame did not work properly")
		}
		t.Log("Testing implant's decryption of LP's rotated transmit message")
		decryptedSrcFrame = UnwrapTransmitFrame(encryptedSrcFrame, testLpReg.CurPubKey, testLpReg.CurKeyPair.PrivKey, testLpReg.XorKey)
		if !bytes.Equal(test.dataframe, decryptedSrcFrame) {
			t.Fatal("[FAIL] Implant's rotated UnwrapTransmitFrame did not work properly")
		}
	}
}

// TODO: add full registration cycle test (sealed frame -> transmit frame with rotations)
