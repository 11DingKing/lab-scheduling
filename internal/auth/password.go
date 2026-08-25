package auth

import "crypto/sha256"
import "encoding/hex"

func HashPassword(password string) string {
	sum := sha256.Sum256([]byte("lab-scheduling:" + password))
	return hex.EncodeToString(sum[:])
}
func CheckPassword(hash, password string) bool { return hash == HashPassword(password) }
