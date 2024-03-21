package shortcode

import (
	"crypto/sha256"
	"math/big"
)

const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
const shortcodeLength = 6

func isUnique(shortcode string, shortcodeCache *map[string]bool) bool {
	 if (*shortcodeCache)[shortcode] {
	   return false
	 }

	return true
}

func storeShortcode(shortcode string, shortcodeCache *map[string]bool) {
  (*shortcodeCache)[shortcode] = true
}

func GenerateShortcode(uuid string, shortcodeCache *map[string]bool) string {
	hash:= sha256.Sum256([]byte(uuid))
	hashInt := new(big.Int).SetBytes(hash[:])

	base := big.NewInt(int64(len(charset)))
	var shortcode string

	for {
		shortcode = ""
		tempHashInt := new(big.Int).Set(hashInt)
		for i := 0; i < shortcodeLength; i++ {
			mod := new(big.Int)
			tempHashInt.DivMod(tempHashInt, base, mod)
			shortcode = string(charset[mod.Int64()]) + shortcode
		}

		if isUnique(shortcode, shortcodeCache) {
			storeShortcode(shortcode, shortcodeCache)
			break
		} else {
			// Modify the hashInt (e.g., by adding 1) to try a different shortcode
			hashInt = hashInt.Add(hashInt, big.NewInt(1))
		}
	}

	return shortcode
}
