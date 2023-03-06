package sessions

import (
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"log"
	"math/rand"
)

// TODO: Move to bcrypt from b64
func SaltedUUID(password string) string {
	saltSize := 32
	bRand := make([]byte, saltSize)
	_, err := rand.Read(bRand[:])
	if err != nil {
		log.Println("Can't read random bytes")
	}
	sha512Hasher := sha512.New()
	bPassword := []byte(password)
	bPassword = append(bPassword, bRand...)
	// TODO: Read again
	sha512Hasher.Write(bPassword)
	sum := sha512Hasher.Sum(nil)
	return base64.StdEncoding.EncodeToString(sum)
}

func MatchHash(uuid string, password []byte, salt []byte) error {
	bRand := []byte(salt)
	bPassword := []byte(password)
	sha512Hasher := sha512.New()
	bPassword = append(bPassword, bRand...)
	// TODO: Read again
	sha512Hasher.Write(bPassword)
	sum := sha512Hasher.Sum(nil)
	encoded := base64.StdEncoding.EncodeToString(sum)
	if encoded != uuid {
		log.Println("Error matching salted password")
		return errors.New("err_salt_pass")
	}
	return nil
}
