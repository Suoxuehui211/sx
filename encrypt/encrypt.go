package encrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

/*CBC加密 按照golang标准库的例子代码
不过里面没有填充的部分,所以补上
*/

//使用PKCS7进行填充，IOS也是7
func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func getKey(key []byte) []byte {
	k := len(key)
	switch k {
	default:
		if (k%16) != 0 && k > 0 {
			a := make([]byte, (16 - k%16))
			key = append(key, a...)
		} else if k == 0 {
			key = make([]byte, 16)
		}
	case 16, 24, 32:
		break
	}

	return key
}

//aes加密，填充秘钥key的16位，24,32分别对应AES-128, AES-192, or AES-256.
func AesCBCEncrypt(rawData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(getKey(key))
	if err != nil {
		return nil, err
	}

	//填充原文
	blockSize := block.BlockSize()
	rawData = PKCS7Padding(rawData, blockSize)
	//初始向量IV必须是唯一，但不需要保密
	cipherText := make([]byte, blockSize+len(rawData))
	//block大小 16
	iv := cipherText[:blockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	//block大小和初始向量大小一定要一致
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText[blockSize:], rawData)

	return cipherText, nil
}

//aes加密，填充秘钥key的16位，24,32分别对应AES-128, AES-192, or AES-256.
func AesCBCEncrypt2(rawData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(getKey(key))
	if err != nil {
		return nil, err
	}

	//填充原文
	blockSize := block.BlockSize()
	rawData = PKCS7Padding(rawData, blockSize)
	//初始向量IV必须是唯一，但不需要保密
	cipherText := make([]byte, blockSize+len(rawData))
	//block大小 16
	iv := cipherText[:blockSize]
	//if _, err := io.ReadFull(rand.Reader,iv); err != nil {
	//	return nil, err
	//}

	//block大小和初始向量大小一定要一致
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText[blockSize:], rawData)

	return cipherText, nil
}

func AesCBCDncrypt(encryptData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()

	if len(encryptData) < blockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := encryptData[:blockSize]
	encryptData = encryptData[blockSize:]

	// CBC mode always works in whole blocks.
	if len(encryptData)%blockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)

	// CryptBlocks can work in-place if the two arguments are the same.
	mode.CryptBlocks(encryptData, encryptData)
	//解填充
	encryptData = PKCS7UnPadding(encryptData)
	return encryptData, nil
}

func Encrypt(rawData, key []byte) (string, error) {
	data, err := AesCBCEncrypt(rawData, key)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(data), nil //StdEncoding
}

func Dncrypt(rawData string, key []byte) (string, error) {
	data, err := base64.StdEncoding.DecodeString(rawData) //StdEncoding
	if err != nil {
		return "", err
	}
	dnData, err := AesCBCDncrypt(data, getKey(key))
	if err != nil {
		return "", err
	}
	return string(dnData), nil
}

func AESBase64Encrypt(originData string, key string) (base64Result string, err error) {
	//iv := []byte{1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6}
	var block cipher.Block
	if block, err = aes.NewCipher(getKey([]byte(key))); err != nil {
		return "", err
	}
	iv := make([]byte, block.BlockSize())
	encrypt := cipher.NewCBCEncrypter(block, iv)
	var source = PKCS7Padding([]byte(originData), block.BlockSize())
	var dst = make([]byte, len(source))
	encrypt.CryptBlocks(dst, source)
	base64Result = base64.StdEncoding.EncodeToString(dst) //RawStdEncoding
	return
}
