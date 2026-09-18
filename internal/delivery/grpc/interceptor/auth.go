// Package grpcauth 提供 gRPC 鉴权拦截器（对齐 HTTP SessionAuth）。
package grpcauth

import (
	"context"
	"strings"

	"github.com/stvenfor/my_go_study/internal/usecase"
	pkgauth "github.com/stvenfor/my_go_study/pkg/auth"
	"github.com/stvenfor/my_go_study/pkg/config"
	jwtmanager "github.com/stvenfor/my_go_study/pkg/jwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ctxKey string

const (
	ContextUserKey        ctxKey = "grpc_user"
	ContextAccessTokenKey ctxKey = "grpc_access_token"
	mdAuthorization              = "authorization"
	mdSessionID                  = "x-session-id"
	mdDeviceID                   = "x-device-id"
)

// Authenticator 校验 access token + device session。
type Authenticator struct {
	local          bool
	jwtMgr         *jwtmanager.Manager
	supabase       config.SupabaseConfig
	deviceSessionUC *usecase.DeviceSessionUsecase
}

// NewLocal 本地 JWT 鉴权。
func NewLocal(jwtMgr *jwtmanager.Manager, sessionUC *usecase.DeviceSessionUsecase) *Authenticator {
	return &Authenticator{local: true, jwtMgr: jwtMgr, deviceSessionUC: sessionUC}
}

// NewSupabase Supabase token 鉴权。
func NewSupabase(cfg config.SupabaseConfig, sessionUC *usecase.DeviceSessionUsecase) *Authenticator {
	return &Authenticator{local: false, supabase: cfg, deviceSessionUC: sessionUC}
}

// UnaryServerInterceptor 一元 RPC 鉴权。
func (a *Authenticator) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "缺少 metadata")
		}
		token := bearerFromMD(md)
		if token == "" {
			return nil, status.Error(codes.Unauthenticated, "未提供 Authorization")
		}

		var user pkgauth.SupabaseUser
		if a.local {
			if a.jwtMgr == nil {
				return nil, status.Error(codes.Internal, "JWT 未配置")
			}
			claims, err := a.jwtMgr.ParseUUID(token)
			if err != nil {
				return nil, status.Error(codes.Unauthenticated, "token 无效")
			}
			user = pkgauth.SupabaseUser{ID: claims.Subject, Email: claims.Email}
		} else {
			u, err := pkgauth.ValidateAccessToken(ctx, a.supabase, token)
			if err != nil {
				return nil, status.Error(codes.Unauthenticated, err.Error())
			}
			user = u
		}

		sessionID := firstMD(md, mdSessionID)
		deviceID := firstMD(md, mdDeviceID)
		if a.deviceSessionUC != nil {
			if err := a.deviceSessionUC.Validate(ctx, user.ID, user.Email, sessionID, deviceID); err != nil {
				if err == usecase.ErrSessionReplaced {
					return nil, status.Error(codes.Unauthenticated, usecase.MsgSessionReplaced)
				}
				return nil, status.Error(codes.Unauthenticated, usecase.MsgSessionInvalid)
			}
		}

		ctx = context.WithValue(ctx, ContextUserKey, user)
		ctx = context.WithValue(ctx, ContextAccessTokenKey, token)
		return handler(ctx, req)
	}
}

// UserFromContext 取出已鉴权用户。
func UserFromContext(ctx context.Context) (pkgauth.SupabaseUser, bool) {
	user, ok := ctx.Value(ContextUserKey).(pkgauth.SupabaseUser)
	return user, ok
}

func bearerFromMD(md metadata.MD) string {
	raw := firstMD(md, mdAuthorization)
	if raw == "" {
		return ""
	}
	const prefix = "bearer "
	if len(raw) > len(prefix) && strings.EqualFold(raw[:len(prefix)], prefix) {
		return strings.TrimSpace(raw[len(prefix):])
	}
	return strings.TrimSpace(raw)
}

func firstMD(md metadata.MD, key string) string {
	vals := md.Get(key)
	if len(vals) == 0 {
		return ""
	}
	return strings.TrimSpace(vals[0])
}
