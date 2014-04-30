package main

import (
  "crypto/aes"
  "crypto/cipher"
  "crypto/rand"
  "encoding/base64"
  "io"
)
type Encryption struct{
  key []byte
}
/**
 * Key only for Testing purpose, remove on production branch.
 * @todo: Remove key from branch.
 */
func encryption() *Encryption {
  e := new(Encryption)
  e.key = []byte("4qyfoZFqH0t9d4Ud3kYv3J1gyjLaQmjq")
  return e
}

func (e Encryption) encrypt(text []byte) []byte {
  block, err := aes.NewCipher(e.key)
  if err != nil {
    panic(err)
  }
  b := encodeBase64(text)
  ciphertext := make([]byte, aes.BlockSize+len(b))
  iv := ciphertext[:aes.BlockSize]
  if _, err := io.ReadFull(rand.Reader, iv); err != nil {
    panic(err)
  }
  cfb := cipher.NewCFBEncrypter(block, iv)
  cfb.XORKeyStream(ciphertext[aes.BlockSize:], b)
  return ciphertext
}

func (e Encryption) decrypt(text []byte) []byte {
  block, err := aes.NewCipher(e.key)
  if err != nil {
    panic(err)
  }
  if len(text) < aes.BlockSize {
    panic("ciphertext too short")
  }
  iv := text[:aes.BlockSize]
  text = text[aes.BlockSize:]
  cfb := cipher.NewCFBDecrypter(block, iv)
  cfb.XORKeyStream(text, text)
  return decodeBase64(text)
}


func encodeBase64(b []byte) []byte {
  return []byte(base64.StdEncoding.EncodeToString(b))
}

func decodeBase64(b []byte) []byte {
  data, err := base64.StdEncoding.DecodeString(string(b))
  if err != nil {
    panic(err)
  }
  return data
}
