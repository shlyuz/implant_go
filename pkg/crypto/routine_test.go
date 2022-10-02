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

func TestRoutineTransmitFrame(t *testing.T) {
	t.Parallel()
	var err error
	impKeyPair, err := asymmetric.GenerateKeypair()
	if err != nil {
		t.Fatal("[FAIL] unable to generate Imp keypair", err)
	}
	for _, test := range testcase {

		encryptedTransmitFrame, lpKeyPair := PrepareTransmitFrame(test.dataframe, impKeyPair.PubKey)
		if bytes.Equal(test.dataframe, encryptedTransmitFrame) {
			t.Fatal("[FAIL] PrepareTransmitFrame did not work properly")
		}
		decryptedTransmitFrame := UnwrapTransmitFrame(encryptedTransmitFrame, impKeyPair.PubKey, lpKeyPair.PrivKey)
		if bytes.Equal(encryptedTransmitFrame, decryptedTransmitFrame) {
			t.Fatal("[FAIL] UnwrapTransmitFrame did not work properly")
		}
		if !bytes.Equal(decryptedTransmitFrame, test.dataframe) {
			t.Fatal("[FAIL] UnwrapTransmitFrame did not produce expected result. Got: ", decryptedTransmitFrame, " Wanted: ", test.dataframe)
		}
	}
}
