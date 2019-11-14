// Package secure encrypting/decrypting the message body.
//
// Copyright 2018 HenryLee. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package secure

import (
	"crypto/aes"
	"fmt"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/utils"
	"github.com/henrylee2cn/goutil"
)

const (
	// SECURE_META_KEY if the metadata is true, perform encryption operation to the body.
	SECURE_META_KEY = "X-Secure" // value: true/false
	// ACCEPT_SECURE_META_KEY if the metadata is true, perform encryption operation to the body.
	ACCEPT_SECURE_META_KEY = "X-Accept-Secure" // value: true/false
)

const (
	// CIPHERVERSION_KEY cipherkey version
	CIPHERVERSION_KEY = "cipherversion"
	// CIPHERTEXT_KEY ciphertext content
	CIPHERTEXT_KEY = "ciphertext"
)

// NewPlugin creates a AES encryption/decryption plugin.
// The cipherkey argument should be the AES key,
// either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func NewPlugin(statCode int32, cipherkey string) erpc.Plugin {
	b := []byte(cipherkey)
	if _, err := aes.NewCipher(b); err != nil {
		erpc.Fatalf("secure: %v", err)
	}
	version := goutil.Md5([]byte(cipherkey))
	return &securePlugin{
		encryptPlugin: &encryptPlugin{
			version:   version,
			cipherkey: b,
			statCode:  statCode,
		},
		decryptPlugin: &decryptPlugin{
			version:   version,
			cipherkey: b,
			statCode:  statCode,
		},
	}
}

// EnforceSecure enforces the body of the encrypted reply message.
// Note: requires that the secure plugin has been registered!
func EnforceSecure(output erpc.Message) {
	output.Meta().Set(SECURE_META_KEY, "true")
}

// WithSecureMeta encrypts the body of the current message.
// Note: requires that the secure plugin has been registered!
func WithSecureMeta() erpc.MessageSetting {
	return func(message erpc.Message) {
		message.Meta().Set(SECURE_META_KEY, "true")
	}
}

// WithAcceptSecureMeta requires the peer to encrypt the replying body.
// Note: requires that the secure plugin has been registered!
func WithAcceptSecureMeta(accept bool) erpc.MessageSetting {
	s := fmt.Sprintf("%v", accept)
	return func(message erpc.Message) {
		message.Meta().Set(ACCEPT_SECURE_META_KEY, s)
	}
}

type swapKey string

const (
	encrypt_rawbody swapKey = ""
	accept_encrypt  swapKey = "0"
)

type (
	securePlugin struct {
		*encryptPlugin
		*decryptPlugin
	}
	encryptPlugin struct {
		version   string
		cipherkey []byte
		statCode  int32
	}
	decryptPlugin encryptPlugin
)

var (
	_ erpc.PreWriteCallPlugin      = (*securePlugin)(nil)
	_ erpc.PreWritePushPlugin      = (*securePlugin)(nil)
	_ erpc.PreWriteReplyPlugin     = (*securePlugin)(nil)
	_ erpc.PreReadCallBodyPlugin   = (*securePlugin)(nil)
	_ erpc.PostReadCallBodyPlugin  = (*securePlugin)(nil)
	_ erpc.PreReadReplyBodyPlugin  = (*securePlugin)(nil)
	_ erpc.PostReadReplyBodyPlugin = (*securePlugin)(nil)
	_ erpc.PreReadPushBodyPlugin   = (*securePlugin)(nil)
	_ erpc.PostReadPushBodyPlugin  = (*securePlugin)(nil)
)

func (e *securePlugin) Name() string {
	return "secure(encrypt/decrypt)"
}

// func (e *decryptPlugin) Name() string {
// 	return "decrypt"
// }

// func (e *encryptPlugin) Name() string {
// 	return "encrypt"
// }

func isSecure(meta *utils.Args) bool {
	b := meta.Peek(SECURE_META_KEY)
	if len(b) == 4 && goutil.BytesToString(b) == "true" {
		return true
	}
	// if the metadata SECURE_META_KEY is not true,
	// do not perform decryption operation to the body!
	if b != nil {
		meta.Del(SECURE_META_KEY)
	}
	return false
}

func (e *encryptPlugin) PreWriteCall(ctx erpc.WriteCtx) *erpc.Status {
	if ctx.Status() != nil {
		return nil
	}
	if !isSecure(ctx.Output().Meta()) {
		_, acceptSecure := ctx.Swap().Load(accept_encrypt)
		if !acceptSecure {
			return nil
		}
		EnforceSecure(ctx.Output())
	}

	// body: perform encryption operation to the body.
	bodyBytes, err := ctx.Output().MarshalBody()
	if err != nil {
		return erpc.NewStatus(e.statCode, "marshal raw body error", err.Error())
	}
	ciphertext := goutil.AESEncrypt(e.cipherkey, bodyBytes)
	ctx.Output().SetBody(&Encrypt{
		Cipherversion: e.version,
		Ciphertext:    goutil.BytesToString(ciphertext),
	})
	return nil
}

func (e *encryptPlugin) PreWritePush(ctx erpc.WriteCtx) *erpc.Status {
	return e.PreWriteCall(ctx)
}

func (e *encryptPlugin) PreWriteReply(ctx erpc.WriteCtx) *erpc.Status {
	return e.PreWriteCall(ctx)
}

func (e *decryptPlugin) PreReadCallBody(ctx erpc.ReadCtx) *erpc.Status {
	b := ctx.PeekMeta(ACCEPT_SECURE_META_KEY)
	accept := goutil.BytesToString(b)
	useDecrypt := isSecure(ctx.Input().Meta())
	if !useDecrypt {
		// if the metadata ACCEPT_SECURE_META_KEY is true,
		// perform encryption operation to the body.
		if accept == "true" {
			ctx.Swap().Store(accept_encrypt, nil)
		}
		return nil
	}
	if accept != "false" {
		ctx.Swap().Store(accept_encrypt, nil)
	}

	// body: to prepare for decryption.
	ctx.Swap().Store(encrypt_rawbody, ctx.Input().Body())
	ctx.Input().SetBody(new(Encrypt))

	return nil
}

func (e *decryptPlugin) PostReadCallBody(ctx erpc.ReadCtx) *erpc.Status {
	rawbody, ok := ctx.Swap().Load(encrypt_rawbody)
	if !ok {
		return nil
	}

	var obj = ctx.Input().Body().(*Encrypt)
	var version = obj.GetCipherversion()
	var bodyBytes []byte
	var err error

	if len(version) > 0 {
		if version != e.version {
			return erpc.NewStatus(
				e.statCode,
				"decrypt ciphertext error",
				fmt.Sprintf("inconsistent encryption version, get:%q, want:%q", obj.GetCipherversion(), e.version),
			)
		}
		ciphertext := obj.GetCiphertext()
		bodyBytes, err = goutil.AESDecrypt(e.cipherkey, goutil.StringToBytes(ciphertext))
		if err != nil {
			return erpc.NewStatus(e.statCode, "decrypt ciphertext error", err.Error())
		}
	}

	ctx.Swap().Delete(encrypt_rawbody)
	ctx.Input().SetBody(rawbody)
	err = ctx.Input().UnmarshalBody(bodyBytes)
	if err != nil {
		return erpc.NewStatus(e.statCode, "unmarshal raw body error", err.Error())
	}
	return nil
}

func (e *decryptPlugin) PreReadReplyBody(ctx erpc.ReadCtx) *erpc.Status {
	return e.PreReadCallBody(ctx)
}

func (e *decryptPlugin) PostReadReplyBody(ctx erpc.ReadCtx) *erpc.Status {
	return e.PostReadCallBody(ctx)
}

func (e *decryptPlugin) PreReadPushBody(ctx erpc.ReadCtx) *erpc.Status {
	return e.PreReadCallBody(ctx)
}

func (e *decryptPlugin) PostReadPushBody(ctx erpc.ReadCtx) *erpc.Status {
	return e.PostReadCallBody(ctx)
}
