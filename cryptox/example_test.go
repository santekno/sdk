package cryptox_test

import (
	"fmt"

	"github.com/santekno/sdk/cryptox"
)

func ExampleEncryptAESGCM() {
	key, _ := cryptox.GenerateAESKey()
	plaintext := []byte("hello, AES-GCM!")

	ciphertext, err := cryptox.EncryptAESGCM(key, plaintext)
	if err != nil {
		panic(err)
	}

	decrypted, err := cryptox.DecryptAESGCM(key, ciphertext)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(decrypted))
	// Output: hello, AES-GCM!
}

func ExampleGenerateAESKey() {
	key, err := cryptox.GenerateAESKey()
	if err != nil {
		panic(err)
	}
	fmt.Println(len(key))
	// Output: 32
}
