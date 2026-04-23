package main

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const (
	serviceName = "goObuAsanaCli"
	accountName = "personal_access_token"
)

var ErrTokenNotFound = errors.New("token not found in keyring")

// TokenStore はクレデンシャルの保存・取得を抽象化し、テスト時のモック化を可能にします。
type TokenStore interface {
	Get() (string, error)
	Set(token string) error
	Delete() error
}

type keyringTokenStore struct{}

func NewTokenStore() TokenStore {
	return &keyringTokenStore{}
}

func (s *keyringTokenStore) Get() (string, error) {
	secret, err := keyring.Get(serviceName, accountName)
	if err != nil {
		return "", ErrTokenNotFound
	}
	return secret, nil
}

func (s *keyringTokenStore) Set(token string) error {
	return keyring.Set(serviceName, accountName, token)
}

func (s *keyringTokenStore) Delete() error {
	return keyring.Delete(serviceName, accountName)
}
