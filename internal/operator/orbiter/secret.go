package orbiter

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

type Secret struct {
	Encryption string
	Encoding   string
	Value      string
	Masterkey  string `yaml:"-"`
}

func (s *Secret) UnmarshalYAML(node *yaml.Node) error {

	type Alias Secret
	alias := &Alias{}
	if err := node.Decode(alias); err != nil {
		return err
	}
	s.Encryption = alias.Encryption
	s.Encoding = alias.Encoding
	if alias.Value == "" {
		return nil
	}

	cipherText, err := base64.URLEncoding.DecodeString(alias.Value)
	if err != nil {
		return err
	}

	if len(s.Masterkey) < 1 || len(s.Masterkey) > 32 {
		return errors.New("Master key size must be between 1 and 32 characters")
	}

	masterKey := make([]byte, 32)
	for idx, char := range []byte(strings.Trim(s.Masterkey, "\n")) {
		masterKey[idx] = char
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return err
	}

	if len(cipherText) < aes.BlockSize {
		return errors.New("Ciphertext block size is too short")
	}

	//IV needs to be unique, but doesn't have to be secure.
	//It's common to put it at the beginning of the ciphertext.
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(cipherText, cipherText)

	if !utf8.Valid(cipherText) {
		return errors.New("Decryption failed")
	}
	//	s.logger.Info("Decoded and decrypted secret")
	s.Value = string(cipherText)
	return nil
}

func (s *Secret) MarshalYAML() (interface{}, error) {

	if s.Value == "" {
		return nil, nil
	}

	if len(s.Masterkey) < 1 || len(s.Masterkey) > 32 {
		return nil, errors.New("Master key size must be between 1 and 32 characters")
	}

	masterKey := make([]byte, 32)
	for idx, char := range []byte(strings.Trim(s.Masterkey, "\n")) {
		masterKey[idx] = char
	}

	c, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, err
	}

	cipherText := make([]byte, aes.BlockSize+len(s.Value))
	iv := cipherText[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(c, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], []byte(s.Value))

	type Alias Secret
	return &Alias{Encryption: "AES256", Encoding: "Base64", Value: base64.URLEncoding.EncodeToString(cipherText)}, nil
}