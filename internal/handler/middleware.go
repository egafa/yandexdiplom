package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/egafa/yandexdiplom/config"
	"github.com/golang-jwt/jwt"
)

type favContextKey string

const (
	authorizationHeader               = "Authorization"
	userCtx             favContextKey = "userId"
)

func UserIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		header := r.Header.Get(authorizationHeader)
		if header == "" {
			http.Error(w, "empty auth header", http.StatusUnauthorized)
			log.Print("empty auth header ")
			return
		}

		headerParts := strings.Split(header, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			http.Error(w, "einvalid auth header", http.StatusUnauthorized)
			log.Print("invalid auth header ", header)
			return
		}

		if len(headerParts[1]) == 0 {
			http.Error(w, "token is empty", http.StatusUnauthorized)
			log.Print("token is empty ", header)
			return
		}

		userId, err := ParseToken(headerParts[1])
		if err != nil {
			http.Error(w, "token parsing error", http.StatusUnauthorized)
			log.Print("token parsing error ", header)
			return
		}

		intID, err := strconv.Atoi(userId)
		if err != nil {
			http.Error(w, "token parsing error convert", http.StatusUnauthorized)
			log.Print("token parsing error convert ", header)
			return
		}

		ctx := context.WithValue(r.Context(), userCtx, &intID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ParseToken(accessToken string) (string, error) {
	tokenClaims := jwt.StandardClaims{}
	token, err := jwt.ParseWithClaims(accessToken, &tokenClaims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}

		return []byte(config.GetSessionKey()), nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*jwt.StandardClaims)
	if !ok {
		return "", errors.New("token claims are not of type *tokenClaims")
	}

	return claims.Id, nil
}
