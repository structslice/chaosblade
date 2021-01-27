/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"

	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/spf13/cobra"
)

type ServerCommand struct {
	baseCommand
}

func (sc *ServerCommand) Init() {
	sc.command = &cobra.Command{
		Use:     "server",
		Short:   "Server mode starts, exposes web services",
		Long:    "Server mode starts, exposes web services. Under the mode, you can send http request to trigger experiments",
		Aliases: []string{"srv"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return spec.ReturnFail(spec.Code[spec.IllegalCommand],
				fmt.Sprintf("less start or stop command"))
		},
		Example: serverExample(),
	}
}

func serverExample() string {
	return `blade server start --port 8000`
}

// add by fuzz
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

//AES加密,CBC
func AesEncrypt(origData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	origData = PKCS7Padding(origData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

//AES解密
func AesDecrypt(crypted, key []byte) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(string(crypted))
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(data))
	blockMode.CryptBlocks(origData, data)
	origData = PKCS7UnPadding(origData)
	return origData, nil
}
