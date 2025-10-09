package txtypes

import "encoding/json"

func IsValidPubKey(bytes []byte) bool {
	if len(bytes) != 40 {
		return false
	}

	return !isZeroByteSlice(bytes)
}

func isZeroByteSlice(bytes []byte) bool {
	for _, s := range bytes {
		if s != 0 {
			return false
		}
	}
	return true
}

// TxSerializable is the interface for serializable transaction types
type TxSerializable interface {
	// This interface is implemented by all transaction types
}

func getTxInfo(tx TxSerializable) (string, error) {
	txInfoBytes, err := json.Marshal(tx)
	if err != nil {
		return "", err
	}
	return string(txInfoBytes), nil
}
