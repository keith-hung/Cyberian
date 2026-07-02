package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// genID returns a random 5-digit numeric ID in the range 10000-99999. The ID
// is not a secret and does not need to be unpredictable; crypto/rand is used
// only to avoid seeding math/rand.
func genID() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(90000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%05d", n.Int64()+10000), nil
}
