// Package keystore is an in-memory key storage for JWT keys.
package keystore

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
)

// key represents a key inside of keystore.
type key struct {
	PrivatePEM string
	PublicPEM  string
}

// KeyStore represents an in-memory key store.
type KeyStore struct {
	store map[string]key
}

// LoadFromFS walks the entire fs finds the keys and stores them into keystore.
func LoadFromFS(fsys fs.FS) (*KeyStore, error) {
	store := make(map[string]key)

	walk := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("path: %s: %w", path, err)
		}

		if d.IsDir() {
			return nil //skip it
		}

		if filepath.Ext(path) != ".pem" {
			return nil //skip
		}

		//pem files
		file, err := fsys.Open(path)
		if err != nil {
			return fmt.Errorf("open: %s: %w", path, err)
		}

		defer file.Close()

		private, err := io.ReadAll(io.LimitReader(file, 1024*1024))
		if err != nil {
			return fmt.Errorf("read: %s: %w", path, err)
		}

		pemBlock, _ := pem.Decode(private)
		if pemBlock == nil || pemBlock.Type != "PRIVATE KEY" {
			return fmt.Errorf("decode private pem: %w", err)
		}

		privateKey, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
		if err != nil {
			return fmt.Errorf("parse private key: %w", err)
		}

		privateRSA, ok := privateKey.(*rsa.PrivateKey)
		if !ok {
			return errors.New("invalid algorithm")
		}

		publicBytes, err := x509.MarshalPKIXPublicKey(&privateRSA.PublicKey)
		if err != nil {
			return fmt.Errorf("marshalling public key: %w", err)
		}

		//into pem block
		publicPemBlock := pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicBytes,
		}

		var builder strings.Builder
		if err := pem.Encode(&builder, &publicPemBlock); err != nil {
			return fmt.Errorf("encoding public into pem: %w", err)
		}

		//use filename as key inside of map
		store[strings.TrimSuffix(path, ".pem")] = key{
			PrivatePEM: string(private),
			PublicPEM:  builder.String(),
		}
		return nil
	}

	if err := fs.WalkDir(fsys, ".", walk); err != nil {
		return nil, fmt.Errorf("walk: %w", err)
	}

	return &KeyStore{store: store}, nil
}

// PrivateKey fetches the private pem for the given key id or returns possible error.
func (ks *KeyStore) PrivateKey(kid string) (string, error) {
	key, ok := ks.store[kid]
	if !ok {
		return "", fmt.Errorf("private key with id %q not found", kid)
	}
	return key.PrivatePEM, nil
}

// PublicKey fetches the public pem for the given key id or returns possible error.
func (ks *KeyStore) PublicKey(kid string) (string, error) {
	key, ok := ks.store[kid]
	if !ok {
		return "", fmt.Errorf("public key with id %q not found", kid)
	}
	return key.PublicPEM, nil
}
