package mid_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"reflect"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/auth"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/errs"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/mid"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
	"github.com/hamidoujand/task-scheduler/business/domain/user/store/memory"
	"github.com/hamidoujand/task-scheduler/foundation/web"
	"golang.org/x/crypto/bcrypt"
)

func TestErrorsMiddleware(t *testing.T) {
	type Response struct {
		Code    int               `json:"code"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields,omitempty"`
	}
	type Data struct {
		input    web.Handler
		hasErr   bool
		response Response
	}
	tests := map[string]Data{
		"handler with no err": {
			input: func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				//no error
				return nil
			},
			hasErr:   false,
			response: Response{Code: http.StatusOK},
		},

		"handler with internal server err": {
			input: func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				return errs.NewAppError(http.StatusInternalServerError, "something is broken in server")
			},
			hasErr: true,
			response: Response{
				Code:    http.StatusInternalServerError,
				Message: http.StatusText(http.StatusInternalServerError),
			},
		},

		"handler with bad request err": {
			input: func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				return errs.NewAppErrorf(http.StatusBadRequest, "task with id %d Not Found", 1)
			},

			hasErr: true,
			response: Response{
				Code:    http.StatusBadRequest,
				Message: "task with id 1 Not Found",
			},
		},

		"handler with failed validation": {
			input: func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				data := struct {
					Name string `json:"name" validate:"required"`
					Age  int    `json:"age" validate:"required"`
				}{}
				v, _ := errs.NewAppValidator()
				fields, _ := v.Check(data)
				return errs.NewAppValidationError(http.StatusBadRequest, "invalid data", fields)
			},
			hasErr: true,
			response: Response{
				Code:    http.StatusBadRequest,
				Message: "invalid data",
				Fields: map[string]string{
					"name": "name is a required field",
					"age":  "age is a required field",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelInfo}))
			middle := mid.Errors(logger)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			wrapped := middle(test.input)

			_ = wrapped(context.Background(), w, req)
			statusCode := w.Result().StatusCode

			if statusCode != test.response.Code {
				t.Errorf("expected the response status to be %d, but got %d", test.response.Code, statusCode)
			}

			if test.hasErr {
				var resp Response
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("should be able to decode response: %s", err)
				}

				if resp.Message != test.response.Message {
					t.Errorf("expected response message to be %q but got %q", test.response.Message, resp.Message)
				}

				if !reflect.DeepEqual(resp.Fields, test.response.Fields) {
					t.Logf("expected\n%+v\ngot\n%+v\n", test.response.Fields, resp.Fields)
					t.Errorf("expected the fields to be the same as well")
				}
			}

		})
	}

}

//==============================================================================

const (
	kid = "s4sKIjD9kIRjxs2tulPqGLdxSfgPErRN1Mu3Hd9k9NQ"
)

//===============================================================================

func TestAuthenticate(t *testing.T) {
	ks := auth.NewMockKeyStore(t)
	usrId := uuid.New()
	now := time.Now()
	pass, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("expected the password to be hashed: %s", err)
	}
	usr := user.User{
		Id:   usrId,
		Name: "John",
		Email: mail.Address{
			Name:    "john",
			Address: "john@gmail.com",
		},
		Roles:        []user.Role{user.RoleUser},
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
		PasswordHash: pass,
	}

	userRepo := memory.Repository{
		Users: map[uuid.UUID]user.User{
			usrId: usr,
		},
	}

	userService := user.NewService(&userRepo)
	a := auth.New(ks, userService)

	tests := []struct {
		name           string
		claims         auth.Claims
		tokenValid     bool
		authHeader     string
		expectError    bool
		expectedStatus int
	}{
		{
			name: "Valid token",
			claims: auth.Claims{
				Roles: []string{user.RoleAdmin.String()},
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   usrId.String(),
					Issuer:    "mid-test",
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
					IssuedAt:  jwt.NewNumericDate(now),
				},
			},
			tokenValid:     true,
			authHeader:     "Bearer ",
			expectError:    false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Without auth headers",
			claims:         auth.Claims{},
			tokenValid:     false,
			authHeader:     "",
			expectError:    true,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid token",
			claims: auth.Claims{
				Roles: []string{user.RoleAdmin.String()},
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   uuid.NewString(),
					Issuer:    "mid-test",
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
					IssuedAt:  jwt.NewNumericDate(now),
				},
			},
			tokenValid:     false,
			authHeader:     "Bearer ",
			expectError:    true,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tkn, err := a.GenerateToken(kid, tt.claims)
			if err != nil {
				t.Fatalf("expected to generate a token: %s", err)
			}

			m := mid.Authenticate(a)
			r := httptest.NewRequest(http.MethodGet, "/tasks", nil)
			w := httptest.NewRecorder()

			if tt.authHeader != "" {
				r.Header.Add("authorization", tt.authHeader+tkn)
			}

			h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				if tt.tokenValid {
					ctxUser, err := auth.GetUser(ctx)
					if err != nil {
						t.Fatalf("expected the user to be inside of ctx: %s", err)
					}

					if ctxUser.Id != usr.Id {
						t.Errorf("user.Id = %s, want %s", ctxUser.Id, usr.Id)
					}

					if !bytes.Equal(usr.PasswordHash, ctxUser.PasswordHash) {
						t.Errorf("user.Password= %s, want %s", string(ctxUser.PasswordHash), string(usr.PasswordHash))
					}

					if tt.claims.Subject != usrId.String() {
						t.Errorf("claims.Subject= %s, want %s", tt.claims.Issuer, usrId)
					}
				}
				return nil
			}

			wrappedHandler := m(h)
			err = wrappedHandler(context.Background(), w, r)
			if (err != nil) != tt.expectError {
				t.Fatalf("expected error: %v, got: %v", tt.expectError, err)
			}

			if err != nil {
				var appErr *errs.AppError
				if !errors.As(err, &appErr) {
					t.Fatalf("expected the error to be an *errs.AppError, got %T", err)
				}

				if appErr.Code != tt.expectedStatus {
					t.Errorf("appErr.Code= %d, want %d", appErr.Code, tt.expectedStatus)
				}
			}
		})
	}
}

func TestAuthorized(t *testing.T) {
	tests := map[string]struct {
		user        user.User
		roles       []user.Role
		expectError bool
		errorCode   int
	}{
		"authorized": {
			user: user.User{
				Roles: []user.Role{user.RoleAdmin},
			},
			roles:       []user.Role{user.RoleAdmin, user.RoleUser},
			expectError: false,
		},

		"unauthorized": {
			user: user.User{
				Roles: []user.Role{user.RoleUser},
			},
			roles:       []user.Role{user.RoleAdmin},
			expectError: true,
			errorCode:   http.StatusUnauthorized,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			//setups
			ks := auth.NewMockKeyStore(t)
			userRepo := memory.Repository{
				Users: map[uuid.UUID]user.User{},
			}
			userService := user.NewService(&userRepo)
			a := auth.New(ks, userService)

			r := httptest.NewRequest(http.MethodGet, "/secret", nil)
			w := httptest.NewRecorder()

			mid := mid.Authorized(a, test.roles...)
			h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

				if !test.expectError {
					u, err := auth.GetUser(ctx)
					if err != nil {
						t.Fatalf("expected authorized user to get into handler: %s", err)
					}
					t.Logf("%+v\n", u)
				}
				return nil
			}
			wrapped := mid(h)
			ctx := auth.SetUser(context.Background(), test.user)

			err := wrapped(ctx, w, r)
			if test.expectError {
				if err == nil {
					t.Fatal("expected to get an error")
				}

				var appErr *errs.AppError
				if !errors.As(err, &appErr) {
					t.Fatalf("expected the type of error to be *errs.AppErr but got %T", err)
				}

				if appErr.Code != test.errorCode {
					t.Errorf("appErr.Code= %d, got %d", test.errorCode, appErr.Code)
				}

			}
		})
	}
}
