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
	"net/url"

	"github.com/henrylee2cn/goutil"
	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/utils"
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

// NewSecurePlugin creates a AES encryption/decryption plugin.
// The cipherkey argument should be the AES key,
// either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func NewSecurePlugin(rerrCode int32, cipherkey string) tp.Plugin {
	b := []byte(cipherkey)
	if _, err := aes.NewCipher(b); err != nil {
		tp.Fatalf("NewSecurePlugin: %v", err)
	}
	version := goutil.Md5([]byte(cipherkey))
	return &securePlugin{
		encryptPlugin: &encryptPlugin{
			version:   version,
			cipherkey: b,
			rerrCode:  rerrCode,
		},
		decryptPlugin: &decryptPlugin{
			version:   version,
			cipherkey: b,
			rerrCode:  rerrCode,
		},
	}
}

// EnforceSecure enforces the body of the encrypted reply message.
// Note: requires that the secure plugin has been registered!
func EnforceSecure(output *tp.Message) {
	output.Meta().Set(SECURE_META_KEY, "true")
}

// WithSecureMeta encrypts the body of the current message.
// Note: requires that the secure plugin has been registered!
func WithSecureMeta() tp.MessageSetting {
	return func(message *tp.Message) {
		message.Meta().Set(SECURE_META_KEY, "true")
	}
}

// WithAcceptSecureMeta requires the peer to encrypt the replying body.
// Note: requires that the secure plugin has been registered!
func WithAcceptSecureMeta(accept bool) tp.MessageSetting {
	s := fmt.Sprintf("%v", accept)
	return func(message *tp.Message) {
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
		rerrCode  int32
	}
	decryptPlugin encryptPlugin
)

var (
	_ tp.PreWriteCallPlugin      = (*encryptPlugin)(nil)
	_ tp.PreWritePushPlugin      = (*encryptPlugin)(nil)
	_ tp.PreWriteReplyPlugin     = (*encryptPlugin)(nil)
	_ tp.PreReadCallBodyPlugin   = (*decryptPlugin)(nil)
	_ tp.PostReadCallBodyPlugin  = (*decryptPlugin)(nil)
	_ tp.PreReadReplyBodyPlugin  = (*decryptPlugin)(nil)
	_ tp.PostReadReplyBodyPlugin = (*decryptPlugin)(nil)
	_ tp.PreReadPushBodyPlugin   = (*decryptPlugin)(nil)
	_ tp.PostReadPushBodyPlugin  = (*decryptPlugin)(nil)
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

func (e *encryptPlugin) PreWriteCall(ctx tp.WriteCtx) *tp.Rerror {
	if ctx.Rerror() != nil {
		return nil
	}
	if !isSecure(ctx.Output().Meta()) {
		_, acceptSecure := ctx.Swap().Load(accept_encrypt)
		if !acceptSecure {
			return nil
		}
		EnforceSecure(ctx.Output())
	}

	// query: perform encryption operation to the query parameters.
	output := ctx.Output()
	// if output.Mtype() != tp.TypeReply {
	u := output.UriObject()
	if len(u.RawQuery) > 0 {
		ciphertext := goutil.AESEncrypt(e.cipherkey, goutil.StringToBytes(u.RawQuery))
		v := make(url.Values, 0)
		v.Set(CIPHERVERSION_KEY, e.version)
		v.Set(CIPHERTEXT_KEY, goutil.BytesToString(ciphertext))
		u.RawQuery = v.Encode()
	}
	// }

	// body: perform encryption operation to the body.
	bodyBytes, err := ctx.Output().MarshalBody()
	if err != nil {
		return tp.NewRerror(e.rerrCode, "marshal raw body error", err.Error())
	}
	ciphertext := goutil.AESEncrypt(e.cipherkey, bodyBytes)
	ctx.Output().SetBody(&Encrypt{
		Cipherversion: e.version,
		Ciphertext:    goutil.BytesToString(ciphertext),
	})
	return nil
}

func (e *encryptPlugin) PreWritePush(ctx tp.WriteCtx) *tp.Rerror {
	return e.PreWriteCall(ctx)
}

func (e *encryptPlugin) PreWriteReply(ctx tp.WriteCtx) *tp.Rerror {
	return e.PreWriteCall(ctx)
}

func (e *decryptPlugin) PreReadCallBody(ctx tp.ReadCtx) *tp.Rerror {
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

	// query: decrypt query parameters
	version := ctx.Query().Get(CIPHERVERSION_KEY)
	if len(version) == 0 {
		return nil
	}
	if version != e.version {
		return tp.NewRerror(
			e.rerrCode,
			"decrypt ciphertext error",
			fmt.Sprintf("inconsistent encryption version, get:%q, want:%q", version, e.version),
		)
	}
	ciphertext := ctx.Query().Get(CIPHERTEXT_KEY)
	queryBytes, err := goutil.AESDecrypt(e.cipherkey, goutil.StringToBytes(ciphertext))
	if err != nil {
		return tp.NewRerror(e.rerrCode, "decrypt ciphertext error", err.Error())
	}
	q := ctx.Query()
	q.Del(CIPHERVERSION_KEY)
	q.Del(CIPHERTEXT_KEY)
	ctx.UriObject().RawQuery = goutil.BytesToString(queryBytes)
	last := q.Encode()
	if len(last) > 0 {
		ctx.UriObject().RawQuery += "&" + last
	}
	return nil
}

func (e *decryptPlugin) PostReadCallBody(ctx tp.ReadCtx) *tp.Rerror {
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
			return tp.NewRerror(
				e.rerrCode,
				"decrypt ciphertext error",
				fmt.Sprintf("inconsistent encryption version, get:%q, want:%q", obj.GetCipherversion(), e.version),
			)
		}
		ciphertext := obj.GetCiphertext()
		bodyBytes, err = goutil.AESDecrypt(e.cipherkey, goutil.StringToBytes(ciphertext))
		if err != nil {
			return tp.NewRerror(e.rerrCode, "decrypt ciphertext error", err.Error())
		}
	}

	ctx.Swap().Delete(encrypt_rawbody)
	ctx.Input().SetBody(rawbody)
	err = ctx.Input().UnmarshalBody(bodyBytes)
	if err != nil {
		return tp.NewRerror(e.rerrCode, "unmarshal raw body error", err.Error())
	}
	return nil
}

func (e *decryptPlugin) PreReadReplyBody(ctx tp.ReadCtx) *tp.Rerror {
	return e.PreReadCallBody(ctx)
}

func (e *decryptPlugin) PostReadReplyBody(ctx tp.ReadCtx) *tp.Rerror {
	return e.PostReadCallBody(ctx)
}

func (e *decryptPlugin) PreReadPushBody(ctx tp.ReadCtx) *tp.Rerror {
	return e.PreReadCallBody(ctx)
}

func (e *decryptPlugin) PostReadPushBody(ctx tp.ReadCtx) *tp.Rerror {
	return e.PostReadCallBody(ctx)
}
