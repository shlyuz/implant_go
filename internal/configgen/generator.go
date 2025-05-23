package configgen

import (
	"bytes"
	"crypto/rand"
	"encoding/hex" // Standard hex for symmetric key
	"fmt"
	"log"

	"shlyuz/pkg/crypto/asymmetric"
	shlyuzHex "shlyuz/pkg/encoding/hex" // Custom Shlyuz hex encoder

	"github.com/BurntSushi/toml"
	"github.com/google/uuid" // For generating unique IDs
)

// --- Struct Definitions (from previous step, now being applied) ---

// CryptoSectionLp defines the [crypto] section for the Listening Post
type CryptoSectionLp struct {
	ImpPk   string `toml:"imp_pk"`
	SymKey  string `toml:"sym_key"`
	XorKey  string `toml:"xor_key"`
	PrivKey string `toml:"priv_key"`
}

// LpSectionDef defines the [lp] section
type LpSectionDef struct {
	ID            string `toml:"id"`
	TransportName string `toml:"transport_name"`
	TaskCheckTime int64  `toml:"task_check_time"`
	InitSignature string `toml:"init_signature"`
}

// LpConfig is the top-level struct for Listening Post configuration
type LpConfig struct {
	Lp     LpSectionDef    `toml:"lp"`
	Crypto CryptoSectionLp `toml:"crypto"`
}

// CryptoSectionImplant defines the [crypto] section for the Implant
type CryptoSectionImplant struct {
	LpPk    string `toml:"lp_pk"`
	SymKey  string `toml:"sym_key"`
	XorKey  string `toml:"xor_key"`
	PrivKey string `toml:"priv_key"`
}

// VzhivlyatSectionDef defines the [vzhivlyat] section for the Implant
type VzhivlyatSectionDef struct {
	ID            string `toml:"id"`
	TransportName string `toml:"transport_name"`
	TaskCheckTime int64  `toml:"task_check_time"`
	InitSignature string `toml:"init_signature"`
}

// ImplantConfig is the top-level struct for Implant configuration
type ImplantConfig struct {
	Vzhivlyat VzhivlyatSectionDef  `toml:"vzhivlyat"`
	Crypto    CryptoSectionImplant `toml:"crypto"`
}

// --- Config Generation Function ---

// GenerateFullConfigs creates new cryptographic keys and parameters for both LP and Implant,
// and marshals them into TOML formatted strings.
func GenerateFullConfigs() (lpConfigToml string, implantConfigToml string, err error) {
	// 1. Generate LP Asymmetric Key Pair
	lpKeyPair, err := asymmetric.GenerateKeypair()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate LP asymmetric key pair: %w", err)
	}
	encodedLpPk := string(shlyuzHex.Encode(lpKeyPair.PubKey[:]))
	encodedLpPrivKey := string(shlyuzHex.Encode(lpKeyPair.PrivKey[:]))

	// 2. Generate Implant Asymmetric Key Pair
	impKeyPair, err := asymmetric.GenerateKeypair()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate Implant asymmetric key pair: %w", err)
	}
	encodedImpPk := string(shlyuzHex.Encode(impKeyPair.PubKey[:]))
	encodedImpPrivKey := string(shlyuzHex.Encode(impKeyPair.PrivKey[:]))

	// 3. Generate Shared 32-byte Symmetric Key
	symKeyBytes := make([]byte, 32)
	if _, err := rand.Read(symKeyBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate symmetric key: %w", err)
	}
	encodedSymKey := hex.EncodeToString(symKeyBytes)

	// 4. Generate Shared XOR Key (single byte)
	xorKeyByte := make([]byte, 1)
	if _, err := rand.Read(xorKeyByte); err != nil {
		return "", "", fmt.Errorf("failed to generate XOR key: %w", err)
	}
	xorKeyStr := fmt.Sprintf("0x%02x", xorKeyByte[0])

	// 5. Common values from user example
	sharedInitSignature := "b'\\xde\\xad\\xf0\\x0d'" // TOML string representation of b'\xde\xad\xf0\r'
	sharedTransportName := "file_transport"
	sharedTaskCheckTime := int64(60)

	// 6. Generate Unique IDs
	lpID := uuid.NewString()
	impID := uuid.NewString()

	// 7. Populate LP Config Struct
	lpToml := LpConfig{
		Lp: LpSectionDef{
			ID:            lpID,
			TransportName: sharedTransportName,
			TaskCheckTime: sharedTaskCheckTime,
			InitSignature: sharedInitSignature,
		},
		Crypto: CryptoSectionLp{
			ImpPk:   encodedImpPk,
			SymKey:  encodedSymKey,
			XorKey:  xorKeyStr,
			PrivKey: encodedLpPrivKey,
		},
	}

	// 8. Populate Implant Config Struct
	implantToml := ImplantConfig{
		Vzhivlyat: VzhivlyatSectionDef{
			ID:            impID,
			TransportName: sharedTransportName,
			TaskCheckTime: sharedTaskCheckTime,
			InitSignature: sharedInitSignature,
		},
		Crypto: CryptoSectionImplant{
			LpPk:    encodedLpPk,
			SymKey:  encodedSymKey,
			XorKey:  xorKeyStr,
			PrivKey: encodedImpPrivKey,
		},
	}

	// 9. Marshal LP Config to TOML
	var lpTomlBuffer bytes.Buffer
	encoderLp := toml.NewEncoder(&lpTomlBuffer)
	if err := encoderLp.Encode(lpToml); err != nil {
		return "", "", fmt.Errorf("failed to marshal LP config to TOML: %w", err)
	}

	// 10. Marshal Implant Config to TOML
	var impTomlBuffer bytes.Buffer
	encoderImp := toml.NewEncoder(&impTomlBuffer)
	if err := encoderImp.Encode(implantToml); err != nil {
		return "", "", fmt.Errorf("failed to marshal Implant config to TOML: %w", err)
	}

	return lpTomlBuffer.String(), impTomlBuffer.String(), nil
}

// --- Main function for CLI (incorporating previous deferred step) ---
func main() {
	lpConf, impConf, err := GenerateFullConfigs()
	if err != nil {
		log.Fatalf("Error generating configs: %v", err)
	}
	fmt.Println("--- Listening Post Config ---")
	fmt.Print(lpConf)
	fmt.Println("\n--- Implant Config ---")
	fmt.Print(impConf)
}
