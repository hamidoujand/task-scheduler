package keystore_test

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/foundation/keystore"
)

const privatePEM = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC+JhRcgbsrdp/Y
HuYuxZFr/Zz7i2l6i9tNPwYsZSnqmf1wRILF6tfqw3Bhsp0/VJe9Bd5cJxcQDaGc
A+c6dmRo68dHNYbMTm50DhmsFXUsiFXjEZlgakvNksGyUPVR62wsRRRD7QOFWKvZ
1qu5uf5rxMWwSOjeeC8r2GQbWl8FBbYa3mWReIMXtSxxn8ciiE5FPR0tDM52Mnll
82gcqRjJtFvvAMRl9xPwuWmvqOpU93Gap93Nh+VkimigxfyjcKArdXtpI5HPLYgL
KqSun1X89wqQdO+auCPwwOJvrpWeHqx7Ait3Nei9bp3sD402i4xgEIVnt+iST/0U
BYqfhIKVAgMBAAECggEAUYMFa3p5f/JEJ2NnjVlIM6DucK6cstnNUtnXjaR4SYdl
q4DTBoDbulm5jUgCPKnIulEPtzVSn6EYTEcoElm6RXf9XztE48QjeUCqJKi2KDbx
int64mfuwttMiWJHJ6ziHEAopc2umrUeRi3OQ7nFpPEfRaibmvKuVf9XOpqM7Rt+
ZLhz7OqCPSFTzpEZZ7MrSTZYcRWvmtw25AGu/J8T4ExewUqBkSaD6XNhTafTu5d2
A5a7r/wnKCP1w3c1Gi0JSPtrlVBlUpWk9tCpS0ds/fT6O1oPvqXspHrBrnD+j1Bx
DzCRZPMRg6h5O2qFUrLWwdvHcFfIQ9QC6nRZOsC9AQKBgQDVFn2R1d4joSs5FWVA
9cXf23ZzQmqkz5FrLwvdMrBowL8kftHYEG+H5m2MAr/NBzs+OtIh8BYQh5EYqHL/
Dkgdm2kckH7wyc6UfJq9k8F23/7OPdsYvjgEkLDTn+2KSQSS9YOIa/YX8aEPiZme
xYL2rgsYgS+o1s0SRsprZbZDNQKBgQDkcP0dYkfXeZFur8ejDa5yv0VKetfPNz4P
6EdSXEZt0A8DTQjveMsPKoFrwz5bLmV0m84WDTsXw0RhTokcu+b/azXKKfFiidFn
00D4p90DNA94XpcKfCaAMtESiWIMg6NmYK88hGIuz1IFrvOHNOjJs6y7pcmdZFKd
iyx46rrN4QKBgDQewiwPobwZSdc2koOnGfU9WuWqUydo1erfoQlDwr58lsQ4eN9e
dclJ5XWfnoZpxGXeQVOnw93bKvRbD3WvaphDURx5g3MmCW9sYvUH1QRcmZicrKCK
tmz3byj0L0fpwEKp5rhRn+oPYhPI1lhtezEXNQOTZbLoh1R3GD/YqxIZAoGBANPK
TWDotWJ4OvU70wLAtHN+EWez7FEZDlkBKN6a3lEBDGorCZW7j8dHySV3pmAy66zo
pnCbY6XsS4FLpqMVMlyrsPr1V+3biGGR4jKmrqlBovYd/DqkT62bb2qYJGclxGAu
U0jwE3cCjzDlurInw4r9Ia/3TKy3TkDxvxF7ziUBAoGAMaQfJJeEqbGvB5I7t8q6
+/eSI6iiW52GfJo4eJCX8fIBoEZh5I7JjyMdV37yRuuweCa3CPdzHMDpfNlrIfew
n895WVxgzRBVOe68T5/LCUjAk/3bY3z0QACsq3iKxBgWo6pOZiqDH1eLJSkMWrXf
nTJUkEKSxx2LlX5gFJVK4PA=
-----END PRIVATE KEY-----`

func TestKeystore(t *testing.T) {
	//setup
	file := fstest.MapFile{
		Data: []byte(privatePEM),
	}

	kid := uuid.NewString()
	filename := kid + ".pem"

	fsys := fstest.MapFS{
		filename: &file,
	}

	//test
	ks, err := keystore.LoadFromFS(fsys)
	if err != nil {
		t.Fatalf("expected the keys to be loaded: %s", err)
	}

	privatePem, err := ks.PrivateKey(kid)
	if err != nil {
		t.Fatalf("expected to get back the private pem with id %s: %s", kid, err)
	}
	if privatePem != privatePEM {
		t.Errorf("privatePEM= %s, got %s", privatePEM, privatePem)
	}

	fetchedPublic, err := ks.PublicKey(kid)
	if err != nil {
		t.Fatalf("expected to fetch public key: %s", err)
	}

	//now generate one public key from private one
	generatedPublic, err := generatePublic(privatePem)
	if err != nil {
		t.Fatalf("expected to generate public from private: %s", err)
	}

	//equal
	if fetchedPublic != generatedPublic {
		t.Errorf("public= %s, got %s", generatedPublic, fetchedPublic)
	}
}

func generatePublic(privatePEM string) (string, error) {
	//turn into pem block
	block, _ := pem.Decode([]byte(privatePEM))
	if block == nil || block.Type != "PRIVATE KEY" {
		return "", errors.New("decoding private pem")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}

	privateRSA, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return "", errors.New("invalid algorithm")
	}
	publicBytes, err := x509.MarshalPKIXPublicKey(&privateRSA.PublicKey)
	if err != nil {
		return "", fmt.Errorf("marshalling public key: %w", err)
	}

	//into pem block
	publicPemBlock := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicBytes,
	}

	var builder strings.Builder
	if err := pem.Encode(&builder, &publicPemBlock); err != nil {
		return "", fmt.Errorf("encoding public into pem: %w", err)
	}
	return builder.String(), nil
}
