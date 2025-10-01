package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/projuktisheba/erp-mini-api/internal/models"
	"github.com/projuktisheba/erp-mini-api/internal/utils"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleManager  Role = "manager"
	RoleEmployee Role = "employee"
)

// consistent context key used everywhere
type contextKey string
const userContextKey = contextKey("user")

// ========================= AUTH USER ==============================
// AuthUser: validates JWT, attaches *models.JWT to context.
// Important: skips OPTIONS (CORS preflight) so preflight won't be blocked.
func (app *application) AuthUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow preflight through
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		app.infoLog.Printf("AuthUser: %s %s Authorization=%q", r.Method, r.URL.Path, authHeader)

		if authHeader == "" {
			app.errorLog.Println("AuthUser: no Authorization header")
			utils.WriteJSON(w, http.StatusUnauthorized, models.Response{
				Error:   true,
				Message: "Unauthorized: Missing Authorization header",
			})
			return
		}

		parts := strings.Fields(authHeader)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			app.errorLog.Println("AuthUser: invalid Authorization header format")
			utils.WriteJSON(w, http.StatusUnauthorized, models.Response{
				Error:   true,
				Message: "Unauthorized: Invalid Authorization header",
			})
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// verify signing method is HMAC (HS256 etc.)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				app.errorLog.Printf("AuthUser: unexpected signing method: %T", token.Method)
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(app.config.JWT.SecretKey), nil
		})

		if err != nil {
			app.errorLog.Printf("AuthUser: token parse error: %v", err)
			utils.WriteJSON(w, http.StatusUnauthorized, models.Response{
				Error:   true,
				Message: "Unauthorized: Invalid token",
			})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			app.errorLog.Println("AuthUser: invalid token claims or token not valid")
			utils.WriteJSON(w, http.StatusUnauthorized, models.Response{
				Error:   true,
				Message: "Unauthorized: Invalid token",
			})
			return
		}

		// optional: explicit exp check
		if expVal, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(expVal) {
				app.errorLog.Println("AuthUser: token expired")
				utils.WriteJSON(w, http.StatusUnauthorized, models.Response{
					Error:   true,
					Message: "Unauthorized: Token expired",
				})
				return
			}
		}

		// Safely map claims to your models.JWT
		tokenUser := &models.JWT{}
		if idf, ok := claims["id"].(float64); ok {
			tokenUser.ID = int64(idf)
		}
		if name, ok := claims["name"].(string); ok {
			tokenUser.Name = name
		}
		if username, ok := claims["username"].(string); ok {
			tokenUser.Username = username
		}
		if role, ok := claims["role"].(string); ok {
			tokenUser.Role = role
		}
		if expf, ok := claims["exp"].(float64); ok {
			tokenUser.ExpiresAt = int64(expf)
		}
		if iatf, ok := claims["iat"].(float64); ok {
			tokenUser.IssuedAt = int64(iatf)
		}

		app.infoLog.Printf("AuthUser: success id=%d username=%s role=%s", tokenUser.ID, tokenUser.Username, tokenUser.Role)

		// attach user to context using consistent key
		ctx := context.WithValue(r.Context(), userContextKey, tokenUser)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ========================= CONTEXT HELPERS ==============================
func (app *application) UserFromContext(ctx context.Context) (*models.JWT, bool) {
	u, ok := ctx.Value(userContextKey).(*models.JWT)
	if !ok || u == nil {
		return nil, false
	}
	return u, true
}

// ========================= ACCESS CONTROL ==============================
func HasAccess(userRole, required Role) bool {
	switch required {
	case RoleAdmin:
		return userRole == RoleAdmin
	case RoleManager:
		return userRole == RoleAdmin || userRole == RoleManager
	case RoleEmployee:
		return true
	default:
		return false
	}
}

func (app *application) RequireRole(required Role) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := app.UserFromContext(r.Context())
			app.infoLog.Printf("RequireRole: required=%s userFound=%v", required, ok)
			if !ok {
				utils.WriteJSON(w, http.StatusUnauthorized, models.Response{
					Error:   true,
					Message: "Unauthorized: No user in context",
				})
				return
			}

			userRole := Role(user.Role)
			app.infoLog.Printf("RequireRole: userRole=%s required=%s", userRole, required)
			if !HasAccess(userRole, required) {
				utils.WriteJSON(w, http.StatusForbidden, models.Response{
					Error:   true,
					Message: "Forbidden: Insufficient permissions",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ========================= LOGGER ==============================

func (app *application) Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.infoLog.Println("Received request:", r.Method, r.URL.Path, "from", r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}