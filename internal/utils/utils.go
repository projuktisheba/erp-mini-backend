package utils

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/projuktisheba/erp-mini-api/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// readJSON read json from request body into data. It accepts a sinle JSON of 1MB max size value in the body
func ReadJSON(w http.ResponseWriter, r *http.Request, data any) error {
	maxBytes := 1048576 //maximum allowable bytes is 1MB

	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	err := dec.Decode(data)
	if err != nil {
		return err
	}

	err = dec.Decode(&struct{}{})

	if err != io.EOF {
		return errors.New("body must only have a single JSON value")
	}

	return nil
}

// writeJSON writes arbitrary data out as json
func WriteJSON(w http.ResponseWriter, status int, data any, headers ...http.Header) error {
	out, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	//add the headers if exists
	if len(headers) > 0 {
		for i, v := range headers[0] {
			w.Header()[i] = v
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(out)
	return nil
}

// badRequest sends a JSON response with the status http.StatusBadRequest, describing the error
func BadRequest(w http.ResponseWriter, err error) {
	var payload struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}

	payload.Error = true
	payload.Message = err.Error()
	_ = WriteJSON(w, http.StatusOK, payload)
}
// NotFound sends a 404 JSON response with a standard structure.
func NotFound(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Resource not found"
	}

	resp := struct {
		Error   bool   `json:"error"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}{
		Error:   true,
		Status:  "not_found",
		Message: message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(resp)
}
// ServerError sends a 500 JSON response with a standard structure.
func ServerError(w http.ResponseWriter, err error) {
	message := "Internal server error"
	if err != nil && err.Error() != "" {
		message = err.Error()
	}

	resp := struct {
		Error   bool   `json:"error"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}{
		Error:   true,
		Status:  "server_error",
		Message: message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(resp)
}
// EnsureDir checks if a directory exists, and creates it if it does not.
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, os.ModePerm)
}

// GenerateJWT generates a JWT token for the given user
func GenerateJWT(user models.JWT, cfg models.JWTConfig) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"id":         user.ID,
		"name":       user.Name,
		"username":   user.Username,
		"role":       user.Role,
		"iss":        cfg.Issuer,
		"aud":        cfg.Audience,
		"exp":        now.Add(cfg.Expiry).Unix(),
		"iat":        now.Unix(),
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod(cfg.Algorithm), claims)
	return token.SignedString([]byte(cfg.SecretKey))
}

// ParseJWT validates the token and returns claims
func ParseJWT(tokenString string, cfg models.JWTConfig) (*models.JWT, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != cfg.Algorithm {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(cfg.SecretKey), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	return &models.JWT{
		ID:        int(claims["id"].(float64)),
		Name:      claims["name"].(string),
		Username:  claims["username"].(string),
		Role:      claims["role"].(string),
		Issuer:    claims["iss"].(string),
		Audience:  claims["aud"].(string),
		ExpiresAt: int64(claims["exp"].(float64)),
		IssuedAt:  int64(claims["iat"].(float64)),
	}, nil
}

// HashPassword generates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// CheckPassword compares a plain password with its hashed version
func CheckPassword(password, hashed string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
	return err == nil
}

// Today returns the current date with time set to 00:00:00
func Today() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}
// NullableTime converts zero time to nil
func NullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

