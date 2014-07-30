package bip32

import (
	"bytes"
	"code.google.com/p/go.crypto/ripemd160"
	"crypto/sha256"
	"encoding/binary"
	"github.com/cmars/basen"
	"github.com/mndrix/btcutil"
	"io"
	"math/big"
	"errors"
)

var (
	curve = btcutil.Secp256k1()
	curveParams = curve.Params()
	BitcoinBase58Encoding = basen.NewEncoding("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")
)

func Sha256(data []byte) []byte {
	hasher := sha256.New()
	hasher.Write(data)
	return hasher.Sum(nil)
}

func DoubleSha256(data []byte) []byte {
	return Sha256(Sha256(data))
}

func RipeMD160(data []byte) []byte {
	hasher := ripemd160.New()
	io.WriteString(hasher, string(data))
	return hasher.Sum(nil)
}

func Hash160(data []byte) []byte {
	return RipeMD160(Sha256(data))
}

func Checksum(data []byte) []byte {
	return DoubleSha256(data)[:4]
}

func AddChecksum(data []byte) []byte {
	checksum := Checksum(data)
	return append(data, checksum...)
}

func Base58Encode(data []byte) []byte {
	return []byte(BitcoinBase58Encoding.EncodeToString(data))
}

func publicKeyForPrivateKey(key []byte) []byte {
	return compressPublicKey(curve.ScalarBaseMult([]byte(key)))
}

func addPublicKeys(key1 []byte, key2 []byte) []byte {
	x1, y1 := expandPublicKey(key1)
	x2, y2 := expandPublicKey(key2)
	return compressPublicKey(curve.Add(x1, y1, x2, y2))
}

func addPrivateKeys(key1 []byte, key2 []byte) []byte {
	var key1Int big.Int
	var key2Int big.Int
	key1Int.SetBytes(key1)
	key2Int.SetBytes(key2)

	key1Int.Add(&key1Int, &key2Int)
	key1Int.Mod(&key1Int, curve.Params().N)

	return key1Int.Bytes()
}

func compressPublicKey(x *big.Int, y *big.Int) []byte {
	var key bytes.Buffer

	// Write header; 0x2 for even y value; 0x3 for odd
	key.WriteByte(byte(0x2) + byte(y.Bit(0)))

	// Write X coord; Pad the key so x is aligned with the LSB. Pad size is key length - header size (1) - xBytes size
	xBytes := x.Bytes()
	for i := 0; i < (PublicKeyCompressedLength - 1 - len(xBytes)); i++ {
		key.WriteByte(0x0)
	}
	key.Write(xBytes)

	return key.Bytes()
}

// As described at https://bitcointa.lk/threads/compressed-keys-y-from-x.95735/
func expandPublicKey(key []byte) (*big.Int, *big.Int) {
	var Y *big.Int
	var X *big.Int
	var qPlus1Div4 *big.Int
	X.SetBytes(key[1:])

	// y^2 = x^3 + ax^2 + b
	// a = 0
	// => y^2 = x^3 + b
	ySquared := X.Exp(X, big.NewInt(3), nil)
	ySquared.Add(ySquared, curveParams.B)

	qPlus1Div4.Add(curveParams.P, big.NewInt(1))
	qPlus1Div4.Div(qPlus1Div4, big.NewInt(4))

	// sqrt(n) = n^((q+1)/4) if q = 3 mod 4
	Y.Exp(ySquared, qPlus1Div4, curveParams.P)

	if uint32(key[0]) % 2 == 0 {
		Y.Sub(curveParams.P, Y)
	}

	return X, Y
}

func uint32Bytes(i uint32) []byte {
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, i)
	return bytes
}

func validatePrivateKey(key []byte) error {
	keyInt, _ := binary.ReadVarint(bytes.NewBuffer(key))
	if keyInt == 0 || bytes.Compare(key, curveParams.N.Bytes()) >= 0 {
		return errors.New("Invalid seed")
	}

	return nil
}

func validateChildPublicKey(key []byte) error {
	x, y := expandPublicKey(key)

	if x.Sign() == 0 || y.Sign() == 0 {
		return errors.New("Invalid public key")
	}

	return nil
}
