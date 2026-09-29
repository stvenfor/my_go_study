package supabase

import "github.com/supabase-community/gotrue-go/types"

func accessTokenSession(accessToken string) types.Session {
	return types.Session{AccessToken: accessToken}
}
