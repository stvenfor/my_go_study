package usecase

import "testing"

func TestMapSupabaseAuthError(t *testing.T) {
	tests := []struct {
		name    string
		errMsg  string
		wantErr error
	}{
		{
			name:    "invalid credentials",
			errMsg:  `response status code 400: {"error":"invalid_grant","error_description":"Invalid login credentials"}`,
			wantErr: ErrInvalidCredentials,
		},
		{
			name:    "user exists",
			errMsg:  `response status code 422: {"msg":"User already registered"}`,
			wantErr: ErrUserExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapSupabaseAuthError(&fakeError{msg: tt.errMsg})
			if got != tt.wantErr {
				t.Fatalf("expected %v, got %v", tt.wantErr, got)
			}
		})
	}
}

type fakeError struct {
	msg string
}

func (e *fakeError) Error() string { return e.msg }
