package firebase

import (
	"fmt"
	"time"

	"github.com/SermoDigital/jose/crypto"
	"github.com/SermoDigital/jose/jws"
	"github.com/SermoDigital/jose/jwt"
)

// clientCertURL is the URL containing the public keys for the Google certs
// (whose private keys are used to sign Firebase Auth ID Tokens).
const clientCertURL = "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"

// defaultAcceptableExpSkew is the default expiry leeway.
const defaultAcceptableExpSkew = 300 * time.Second

// VerifyIDToken parses and verifies a Firebase ID Token.
// A Firebase application can identify itself to a trusted backend server by
// sending its Firebase ID Token (accessible via the getToken API in the
// Firebase Authentication client) with its request.
// The backend server can then use the VerifyIDToken() function to verify the
// token is valid, meaning: the token is properly signed, has not expired,
// and it was issued for a given project ID.
func VerifyIDToken(projectID, tokenString string) (*Token, error) {
	decodedJWT, err := jws.ParseJWT([]byte(tokenString))
	if err != nil {
		return nil, err
	}
	decodedJWS, ok := decodedJWT.(jws.JWS)
	if !ok {
		return nil, ErrValue{
			msg: "Firebase Auth ID Token cannot be decoded",
			val: decodedJWT,
		}
	}

	keys := func(j jws.JWS) ([]interface{}, error) {
		certs := &Certificates{URL: clientCertURL}
		kid, ok := j.Protected().Get("kid").(string)
		if !ok {
			return nil, ErrValue{
				msg: "Firebase Auth ID Token has no 'kid' claim",
				val: j.Protected(),
			}
		}
		cert, certErr := certs.Cert(kid)
		if certErr != nil {
			return nil, certErr
		}
		return []interface{}{cert.PublicKey}, nil
	}

	err = decodedJWS.VerifyCallback(keys,
		[]crypto.SigningMethod{crypto.SigningMethodRS256},
		&jws.SigningOpts{Number: 1, Indices: []int{0}})
	if err != nil {
		return nil, err
	}

	ks, _ := keys(decodedJWS)
	key := ks[0]
	if err := decodedJWT.Validate(key, crypto.SigningMethodRS256, validator(projectID)); err != nil {
		return nil, err
	}

	return &Token{delegate: decodedJWT}, nil
}

func validator(projectID string) *jwt.Validator {
	v := &jwt.Validator{}
	v.EXP = defaultAcceptableExpSkew
	v.SetAudience(projectID)
	v.SetIssuer(fmt.Sprintf("https://securetoken.google.com/%s", projectID))
	v.Fn = func(claims jwt.Claims) error {
		subject, ok := claims.Subject()
		if !ok || len(subject) == 0 || len(subject) > 128 {
			return jwt.ErrInvalidSUBClaim
		}
		return nil
	}
	return v
}
