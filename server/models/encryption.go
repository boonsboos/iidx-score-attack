package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"

	"gorm.io/gorm"
	"iidx.boonsboos.nl/server/config"
)

// gorm hooks and utility functions to encrypt sensitive user data before saving to the database and decrypting it when reading from the database.

func (p *Player) BeforeSave(tx *gorm.DB) (err error) {
	if !p.RefreshToken.Valid || p.RefreshToken.String == "" {
		return nil
	}

	encryptedRefreshToken, err := encrypt([]byte(p.RefreshToken.String), []byte(config.ServerConfig.EncryptionKey))
	if err != nil {
		return err
	}

	p.RefreshToken = sql.NullString{
		String: hex.EncodeToString(encryptedRefreshToken),
		Valid:  len(encryptedRefreshToken) != 0,
	}
	return nil
}

func (p *Player) AfterFind(tx *gorm.DB) (err error) {
	if !p.RefreshToken.Valid || p.RefreshToken.String == "" {
		return nil
	}

	token, err := hex.DecodeString(p.RefreshToken.String)
	if err != nil {
		return err
	}

	decryptedRefreshToken, err := decrypt(token, []byte(config.ServerConfig.EncryptionKey))

	if err != nil {
		return err
	}

	p.RefreshToken = sql.NullString{
		String: string(decryptedRefreshToken),
		Valid:  len(decryptedRefreshToken) != 0,
	}
	return nil
}

func encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
