package signer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strconv"
	"time"
)

const (
	SignatureQueryParameter = "sig"
)

type Signer struct {
	key []byte
}

func New(key string) *Signer {
	return &Signer{key: []byte(key)}
}

// SignURL adds a sha256 hmac signature to a URL as a query parameter.
func (signer *Signer) SignURL(u url.URL) url.URL {
	// re-encode the query parameters so they are sorted
	u.RawQuery = u.Query().Encode()
	signature := signer.computeMac(u.String())

	params := u.Query()
	params.Set(SignatureQueryParameter, base64.URLEncoding.EncodeToString(signature))
	u.RawQuery = params.Encode()

	return u
}

// SignURLWithTTL adds a sha256 hmac signature to a URL with a ttl.
func (signer *Signer) SignURLWithTTL(u url.URL, ttl time.Duration) (url.URL, time.Time) {
	expires := time.Now().Add(ttl).UTC()

	pathToSign := url.URL{
		Path: u.Path,
		RawQuery: url.Values{
			"exp": []string{strconv.FormatInt(expires.Unix(), 10)},
		}.Encode(),
	}

	return signer.SignURL(pathToSign), expires
}

// VerifyURL checks if the URL has a valid signature.
func (signer *Signer) VerifyURL(u url.URL) bool {
	params := u.Query()
	sig := params.Get(SignatureQueryParameter)
	if sig == "" {
		return false
	}

	got, err := base64.URLEncoding.DecodeString(sig)
	if err != nil {
		return false
	}

	params.Del(SignatureQueryParameter)
	u.RawQuery = params.Encode()

	want := signer.computeMac(u.String())

	return hmac.Equal(got, want)
}

// VerifyURLAndNotExpired checks if the URL has a valid signature and has not expired.
func (signer *Signer) VerifyURLAndNotExpired(u url.URL) bool {
	urlToVerify := url.URL{
		Path:     u.Path,
		RawQuery: u.RawQuery,
	}

	if !signer.VerifyURL(urlToVerify) {
		return false
	}

	exp := u.Query().Get("exp")
	if exp == "" {
		return false
	}

	expiresUnix, err := strconv.ParseInt(exp, 10, 64)
	if err != nil {
		return false
	}

	return expiresUnix > time.Now().Unix()
}

func (signer *Signer) computeMac(data string) []byte {
	mac := hmac.New(sha256.New, signer.key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
