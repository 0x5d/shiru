package auth

import (
	"context"
	"fmt"

	"google.golang.org/api/idtoken"
)

type GoogleClaims struct {
	Sub       string
	Email     string
	Name      string
	AvatarURL string
}

type GoogleVerifier struct {
	clientID string
}

func NewGoogleVerifier(clientID string) *GoogleVerifier {
	return &GoogleVerifier{clientID: clientID}
}

func (v *GoogleVerifier) Verify(ctx context.Context, credential string) (*GoogleClaims, error) {
	payload, err := idtoken.Validate(ctx, credential, v.clientID)
	if err != nil {
		return nil, fmt.Errorf("validating google id token: %w", err)
	}

	sub, _ := payload.Claims["sub"].(string)
	if sub == "" {
		return nil, fmt.Errorf("missing sub claim in id token")
	}

	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)

	return &GoogleClaims{
		Sub:       sub,
		Email:     email,
		Name:      name,
		AvatarURL: picture,
	}, nil
}
