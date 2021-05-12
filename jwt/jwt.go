// The jwt package creates a Json Web Token (JWT) and also provides validity checks on JWT
// submiited. It has 2 exportable functions.
// 1. MakeJwt - creates a JWT, 2. ChkJwt - vlaidates the JWT
package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// The variables used in the package are the Secret Key, Expiry Time and Payload
// In a Json Web Token there are 3 parts. Header, Payload and Signature
// The secret key variable is used to hash the Header and Payload and store the value into the Signature
// The Secret Key amd Expirty Time are exportable and can be set by the initiating process
var (
	SecretKey string = "@2Aa"
	//	ExpiryTime  time.Duration = 5
	PayLoadData PayLoad
)

// In a Json Web Token there are 3 parts - Header, Payload and Signature
// The Payload rovides more information on the user like name, email, mobile number and ipAddress and expiry of the JWT
// The expiry is based on the ExpiryTime which defaults to 5 mins
// if the expiry exceeds the 5 mins, the token becomes invalid
type PayLoad struct {
	Expiry    int64
	UserId    string
	Name      string
	Sbu       string
	Email     string
	MobileNbr string
	IpAddress string
	Id        int
}

// Base64Encode takes in a string and returns a base 64 encoded string
func base64Encode(src string) string {
	return strings.
		TrimRight(base64.URLEncoding.
			EncodeToString([]byte(src)), "=")
}

// Base64Encode takes in a base 64 encoded string and returns the actual string or an error of it fails to decode the string
func base64Decode(src string) (string, error) {
	if l := len(src) % 4; l > 0 {
		src += strings.Repeat("=", 4-l)
	}
	decoded, err := base64.URLEncoding.DecodeString(src)
	if err != nil {
		errMsg := fmt.Errorf("Decoding Error %s", err)
		return "", errMsg
	}
	return string(decoded), nil
}

// Generates a JWT made up of header, payload and signature and returns the JWT.
// The signature is the hash of the header & payload.
func MakeJwt(expiryTime time.Duration, payload PayLoad) string {

	type Header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}

	header := Header{
		Alg: "HS256",
		Typ: "JWT",
	}

	str, _ := json.Marshal(header)
	header64 := base64Encode(string(str))

	if expiryTime != 0 {
		// Set JWT Expiry in Payload
		payload.Expiry = time.Now().Add(expiryTime * time.Minute).Unix()
	} else {
		// No JWT Expiry
		payload.Expiry = 0
	}

	encodedPayload, _ := json.Marshal(payload)
	signatureValue := header64 + "." + base64Encode(string(encodedPayload))

	strToken := signatureValue + "." + makeHash(signatureValue, SecretKey)

	hexToken := hex.EncodeToString([]byte(strToken))

	return hexToken

}

// Checks if JWT provided is valid
// 1. Ensures JWT has header, payload and signature
// 2. Checks if token has expired
// 3. Checks if signature is valid
// Returns the clear payload and an error if validation fails
func ValidateJwt(jwtHex string, secretKey string) (payLoadReturn PayLoad, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error in Validate JWT : %s", r)
		}
	}()

	var payLoad PayLoad

	jwt, err := hex.DecodeString(jwtHex)
	if err != nil {
		panic(err)
	}

	token := strings.Split(string(jwt), ".")

	// check if the jwt token contains
	// header, payload and token
	if len(token) != 3 {
		err := errors.New("Invalid token: token should contain header, payload and secret")
		return payLoad, err
	}

	// decode payload
	decodedPayload, PayloadErr := base64Decode(token[1])
	if PayloadErr != nil {
		return payLoad, fmt.Errorf("Invalid payload: %s", PayloadErr.Error())
	}

	payLoad = PayLoad{}

	// parses payload from string to a struct
	ParseErr := json.Unmarshal([]byte(decodedPayload), &payLoad)
	if ParseErr != nil {
		return payLoad, fmt.Errorf("Invalid payload: %s", ParseErr.Error())
	}

	// checks if the token has expired.
	if payLoad.Expiry != 0 {
		if time.Now().Unix() > payLoad.Expiry {
			return payLoad, errors.New("Expired token: token has expired")
		}
	}

	// check Hash of the Header & Payload is Ok
	if chkHash(token[0], token[1], token[2], secretKey) == false {
		return payLoad, errors.New("Invalid token")
	}

	return payLoad, nil

}

// use secret key to recreate same hash  of header & payload and see if it is equal to the signature
func chkHash(token0, token1, token2, secret string) bool {

	if makeHash(token0+"."+token1, secret) == token2 {
		return true
	} else {
		return false
	}

}

// Hash generates a Hmac256 hash of a string using a secret
func makeHash(src string, secretKey string) string {
	key := []byte(secretKey)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(src))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
