package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// StreamEncryptor provides streaming encryption capabilities
type StreamEncryptor struct {
	gcm cipher.AEAD
	key []byte
}

// NewStreamEncryptor creates a new stream encryptor with the given key
func NewStreamEncryptor(key []byte) (*StreamEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes for AES-256, got %d bytes", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	return &StreamEncryptor{
		gcm: gcm,
		key: key,
	}, nil
}

// EncryptStream encrypts data from reader and writes to writer
func (se *StreamEncryptor) EncryptStream(reader io.Reader, writer io.Writer) error {
	// Generate a random nonce
	nonce := make([]byte, se.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Write nonce to output first
	if _, err := writer.Write(nonce); err != nil {
		return fmt.Errorf("failed to write nonce: %w", err)
	}

	// Create a buffer for reading chunks
	const chunkSize = 64 * 1024 // 64KB chunks
	buffer := make([]byte, chunkSize)

	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read data: %w", err)
		}

		// Encrypt the chunk
		chunk := buffer[:n]
		encrypted := se.gcm.Seal(nil, nonce, chunk, nil)

		// Write encrypted chunk size first (for decryption)
		sizeBytes := make([]byte, 4)
		sizeBytes[0] = byte(len(encrypted) >> 24)
		sizeBytes[1] = byte(len(encrypted) >> 16)
		sizeBytes[2] = byte(len(encrypted) >> 8)
		sizeBytes[3] = byte(len(encrypted))

		if _, err := writer.Write(sizeBytes); err != nil {
			return fmt.Errorf("failed to write chunk size: %w", err)
		}

		// Write encrypted chunk
		if _, err := writer.Write(encrypted); err != nil {
			return fmt.Errorf("failed to write encrypted chunk: %w", err)
		}

		// Increment nonce for next chunk (simple counter mode)
		for i := len(nonce) - 1; i >= 0; i-- {
			nonce[i]++
			if nonce[i] != 0 {
				break
			}
		}
	}

	return nil
}

// StreamDecryptor provides streaming decryption capabilities
type StreamDecryptor struct {
	gcm cipher.AEAD
	key []byte
}

// NewStreamDecryptor creates a new stream decryptor with the given key
func NewStreamDecryptor(key []byte) (*StreamDecryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes for AES-256, got %d bytes", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	return &StreamDecryptor{
		gcm: gcm,
		key: key,
	}, nil
}

// DecryptStream decrypts data from reader and writes to writer
func (sd *StreamDecryptor) DecryptStream(reader io.Reader, writer io.Writer) error {
	// Read the nonce first
	nonce := make([]byte, sd.gcm.NonceSize())
	if _, err := io.ReadFull(reader, nonce); err != nil {
		return fmt.Errorf("failed to read nonce: %w", err)
	}

	originalNonce := make([]byte, len(nonce))
	copy(originalNonce, nonce)

	sizeBytes := make([]byte, 4)

	for {
		// Read chunk size
		n, err := io.ReadFull(reader, sizeBytes)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read chunk size: %w", err)
		}
		if n != 4 {
			return fmt.Errorf("incomplete chunk size read")
		}

		chunkSize := int(sizeBytes[0])<<24 | int(sizeBytes[1])<<16 | int(sizeBytes[2])<<8 | int(sizeBytes[3])

		// Read encrypted chunk
		encryptedChunk := make([]byte, chunkSize)
		if _, err := io.ReadFull(reader, encryptedChunk); err != nil {
			return fmt.Errorf("failed to read encrypted chunk: %w", err)
		}

		// Decrypt the chunk
		decrypted, err := sd.gcm.Open(nil, nonce, encryptedChunk, nil)
		if err != nil {
			return fmt.Errorf("failed to decrypt chunk: %w", err)
		}

		// Write decrypted chunk
		if _, err := writer.Write(decrypted); err != nil {
			return fmt.Errorf("failed to write decrypted chunk: %w", err)
		}

		// Increment nonce for next chunk
		for i := len(nonce) - 1; i >= 0; i-- {
			nonce[i]++
			if nonce[i] != 0 {
				break
			}
		}
	}

	return nil
}
