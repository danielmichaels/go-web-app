package render

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	"github.com/rs/zerolog"
)

func Render(ctx context.Context, w http.ResponseWriter, status int, t templ.Component) error {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "text/html")
	return t.Render(ctx, w)
}

type contextKey string

const authUserContextKey = contextKey("authUser")

type CtxUser struct {
	UserID         string
	TeamID         string
	ProviderUserID string
	UserMetadata   UserMetadata
}
type UserMetadata struct {
	Email  string
	Avatar string
}

func SetUserContext(r *http.Request, ctxUser CtxUser) *http.Request {
	v := &CtxUser{
		UserID:         ctxUser.UserID,
		TeamID:         ctxUser.TeamID,
		ProviderUserID: ctxUser.ProviderUserID,
		UserMetadata: UserMetadata{
			Avatar: ctxUser.UserMetadata.Avatar,
			Email:  ctxUser.UserMetadata.Email,
		},
	}
	ctx := context.WithValue(r.Context(), authUserContextKey, v)
	return r.WithContext(ctx)
}

func GetUserContext(r *http.Request) *CtxUser {
	user, ok := r.Context().Value(authUserContextKey).(*CtxUser)
	if !ok {
		return nil
	}
	return user
}

func (c *CtxUser) MarshalZerologObject(e *zerolog.Event) {
	e.Str("userID", c.UserID).
		Str("teamID", c.TeamID).
		Str("providerUserID", c.ProviderUserID)
}

func GetUserEmail(ctx context.Context) string {
	if u, ok := ctx.Value(authUserContextKey).(*CtxUser); ok {
		return u.UserMetadata.Email
	}
	return ""
}
func GetUserAvatar(ctx context.Context) string {
	defaultAvatar := "/static/img/default-avatar.png"
	if u, ok := ctx.Value(authUserContextKey).(*CtxUser); ok {
		return u.UserMetadata.Avatar
	}
	return defaultAvatar
}
