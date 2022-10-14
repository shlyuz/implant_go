package routine

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"log"
	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/crypto/symmetric"
	shlyuzHex "shlyuz/pkg/encoding/hex"
	"shlyuz/pkg/encoding/xor"
	"shlyuz/pkg/utils/logging"
)

type EncryptedFrame struct {
	Frame_id  int
	Data      []byte
	Chunk_len int
}

func PrepareSealedFrame(dataFrame []byte, lpPubKey asymmetric.PublicKey, xorKey int, initSig []byte) ([]byte, asymmetric.AsymmetricKeyPair) {
	log.SetPrefix(logging.GetLogPrefix())
	symMsg := symmetric.Encrypt(dataFrame)

	var encryptedSymMsg bytes.Buffer
	binary.Write(&encryptedSymMsg, binary.BigEndian, symMsg.Message)
	symMsgFrame := EncryptedFrame{0, encryptedSymMsg.Bytes(), len(encryptedSymMsg.Bytes())}

	chunkFrame, err := json.Marshal(symMsgFrame)
	if err != nil {
		log.Println("invalid dataframe")
	}
	hexChunkFrame := shlyuzHex.Encode(chunkFrame)
	xorHexChunkFrame := xor.XorMessage(hexChunkFrame, xorKey)

	hexKey := make([]byte, hex.EncodedLen(len(symMsg.Key)))
	hex.Encode(hexKey, symMsg.Key)
	hexedHexKey := make([]byte, hex.EncodedLen(len(hexKey)))
	hex.Encode(hexedHexKey, hexKey)

	chunkMsg := make([]byte, len(hexedHexKey)+len(xorHexChunkFrame))
	copy(chunkMsg[:], hexedHexKey)
	copy(chunkMsg[len(hexedHexKey):], xorHexChunkFrame)

	preparedChunkFrame := make([]byte, len(symMsg.Key)+len(chunkMsg))
	copy(preparedChunkFrame[:], symMsg.Key)
	copy(preparedChunkFrame[len(symMsg.Key):], chunkMsg)

	hexedXorHexChunkFrame := shlyuzHex.Encode(preparedChunkFrame)

	ImpKeyPair, err := asymmetric.GenerateKeypair()
	if err != nil {
		log.Println("Unable to generate key pair")
	}

	encBox := new(asymmetric.AsymmetricBox)
	*encBox = asymmetric.EncryptSealed(hexedXorHexChunkFrame, lpPubKey)
	retMsg := make([]byte, len(encBox.Message)+len(encBox.IV))
	copy(retMsg[:], encBox.IV[:])
	copy(retMsg[len(encBox.IV):], encBox.Message)

	initMsg := make([]byte, len(initSig)+len(retMsg))
	copy(initMsg[:len(initSig)], initSig) // prepend the init signature
	copy(initMsg[len(initSig):], retMsg)

	return initMsg, ImpKeyPair
}

func PrepareTransmitFrame(dataFrame []byte, lpPubKey asymmetric.PublicKey, xorKey int) ([]byte, asymmetric.AsymmetricKeyPair) {
	log.SetPrefix(logging.GetLogPrefix())
	symMsg := symmetric.Encrypt(dataFrame)

	var encryptedSymMsg bytes.Buffer
	binary.Write(&encryptedSymMsg, binary.BigEndian, symMsg.Message)
	symMsgFrame := EncryptedFrame{0, encryptedSymMsg.Bytes(), len(encryptedSymMsg.Bytes())}

	chunkFrame, err := json.Marshal(symMsgFrame)
	if err != nil {
		log.Println("invalid dataframe")
	}
	hexChunkFrame := shlyuzHex.Encode(chunkFrame)
	xorHexChunkFrame := xor.XorMessage(hexChunkFrame, xorKey)

	hexKey := make([]byte, hex.EncodedLen(len(symMsg.Key)))
	hex.Encode(hexKey, symMsg.Key)
	hexedHexKey := make([]byte, hex.EncodedLen(len(hexKey)))
	hex.Encode(hexedHexKey, hexKey)

	chunkMsg := make([]byte, len(hexedHexKey)+len(xorHexChunkFrame))
	copy(chunkMsg[:], hexedHexKey)
	copy(chunkMsg[len(hexedHexKey):], xorHexChunkFrame)

	preparedChunkFrame := make([]byte, len(symMsg.Key)+len(chunkMsg))
	copy(preparedChunkFrame[:], symMsg.Key)
	copy(preparedChunkFrame[len(symMsg.Key):], chunkMsg)

	hexedXorHexChunkFrame := shlyuzHex.Encode(preparedChunkFrame)

	ImpKeyPair, err := asymmetric.GenerateKeypair()
	if err != nil {
		log.Println("Unable to generate key pair")
	}

	encBox := new(asymmetric.AsymmetricBox)
	*encBox = asymmetric.Encrypt(hexedXorHexChunkFrame, lpPubKey, ImpKeyPair.PrivKey)
	retMsg := make([]byte, len(encBox.Message)+len(encBox.IV))
	copy(retMsg[:], encBox.IV[:])
	copy(retMsg[len(encBox.IV):], encBox.Message)

	return retMsg, ImpKeyPair
}

func UnwrapTransmitFrame(transmitFrame []byte, peerPubKey asymmetric.PublicKey, myPrivKey asymmetric.PrivateKey, xorKey int) []byte {
	decryptionBox := new(asymmetric.AsymmetricBox)
	decryptionBox.Message = transmitFrame[24:]
	decryptionBox.IV = (*[24]byte)(transmitFrame[:24])

	decBox, boolAsymSuccess := asymmetric.Decrypt(*decryptionBox, peerPubKey, myPrivKey)
	if !boolAsymSuccess {
		log.Println("invalidmyPrivKey transmit frame")
	}
	unhexedMsg := shlyuzHex.Decode(decBox.Message)
	symKey := unhexedMsg[0:16]

	unxorFrame := xor.XorMessage(unhexedMsg[len(symKey):], xorKey) // appendedHexChunkFrame
	// nextSymKey := unxorFrame[:64]
	uncFrame := shlyuzHex.Decode(unxorFrame[64:]) // this is chunkFrame

	var chunks EncryptedFrame
	json.Unmarshal(uncFrame, &chunks)

	dataFrame := chunks.Data
	recvMsg := symmetric.Decrypt(dataFrame, symKey)
	if recvMsg.IsEncrypted {
		log.Println("unable to extract raw message from transmit frame")
	}
	return recvMsg.Message
}

func UnwrapSealedFrame(transmitFrame []byte, myPrivKey asymmetric.PrivateKey, peerPubKey asymmetric.PublicKey, xorKey int, initSig []byte) []byte {
	// Check for init sig and compare against expected. Strip it if it works, otherwise return a null message to signal an error
	retrievedInitSig := make([]byte, len(initSig))
	copy(retrievedInitSig[:], transmitFrame[:len(initSig)])
	sigCheckRes := bytes.Compare(retrievedInitSig, initSig)

	if sigCheckRes != 0 {
		// we didn't get a valid init sign in this message. Return a null and discard outside of routine
		log.Println("invalid init sig, got'", retrievedInitSig, "'")
		return nil
	}

	trueFrameLen := len(transmitFrame) - len(initSig)
	strippedTransmitFrame := make([]byte, trueFrameLen)
	copy(strippedTransmitFrame[:], transmitFrame[len(initSig):])

	decryptionBox := new(asymmetric.AsymmetricBox)
	decryptionBox.Message = strippedTransmitFrame[24:]
	decryptionBox.IV = (*[24]byte)(strippedTransmitFrame[:24])

	decBox, boolAsymSuccess := asymmetric.DecryptSealed(*decryptionBox, myPrivKey, peerPubKey)
	if !boolAsymSuccess {
		log.Println("invalidmyPrivKey transmit frame")
	}
	unhexedMsg := shlyuzHex.Decode(decBox.Message)
	symKey := unhexedMsg[0:16]
	unxorFrame := xor.XorMessage(unhexedMsg[len(symKey):], xorKey)
	// nextSymKey := unxorFrame[:64]
	uncFrame := shlyuzHex.Decode(unxorFrame[64:]) // this is chunkFrame

	var chunks EncryptedFrame
	json.Unmarshal(uncFrame, &chunks)

	dataFrame := chunks.Data
	recvMsg := symmetric.Decrypt(dataFrame, symKey)
	if recvMsg.IsEncrypted {
		log.Println("unable to extract raw message from sealed transmit frame")
	}
	return recvMsg.Message

}
