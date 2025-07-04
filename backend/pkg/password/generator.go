package password

import (
	"crypto/rand"
	"math/big"
)

const (
	// Character sets for password generation
	lowercaseLetters = "abcdefghijklmnopqrstuvwxyz"
	uppercaseLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits           = "0123456789"
	specialChars     = "!@#$%^&*()_+-=[]{}|;:,.<>?"
)

// GenerateTemporaryPassword generates a secure temporary password
func GenerateTemporaryPassword() string {
	// Use all character sets for temporary passwords
	allChars := lowercaseLetters + uppercaseLetters + digits + specialChars
	
	// Generate 12-character password
	password := make([]byte, 12)
	
	// Ensure at least one character from each set
	password[0] = randomChar(lowercaseLetters)
	password[1] = randomChar(uppercaseLetters)
	password[2] = randomChar(digits)
	password[3] = randomChar(specialChars)
	
	// Fill the rest with random characters
	for i := 4; i < 12; i++ {
		password[i] = randomChar(allChars)
	}
	
	// Shuffle the password to avoid predictable patterns
	shuffleBytes(password)
	
	return string(password)
}

// randomChar returns a random character from the given string
func randomChar(s string) byte {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(s))))
	if err != nil {
		panic(err)
	}
	return s[n.Int64()]
}

// shuffleBytes randomly shuffles a byte slice in place
func shuffleBytes(b []byte) {
	for i := len(b) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			panic(err)
		}
		b[i], b[j.Int64()] = b[j.Int64()], b[i]
	}
}