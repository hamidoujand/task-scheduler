package users_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/errs"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/handlers/users"
	"github.com/hamidoujand/task-scheduler/business/domain/user"
	"github.com/hamidoujand/task-scheduler/business/domain/user/store/memory"
	"golang.org/x/crypto/bcrypt"
)

func TestCreateUser(t *testing.T) {
	tests := map[string]struct {
		input         users.NewUser
		expectError   bool
		statusCode    int
		invalidFields []string
	}{
		"success": {
			input: users.NewUser{
				Name:            "John Doe",
				Email:           "john@gmail.com",
				Password:        "test1234",
				PasswordConfirm: "test1234",
				Roles:           []string{"admin"},
			},
			expectError: false,
			statusCode:  http.StatusCreated,
		},
		"duplicated email": {
			input: users.NewUser{
				Name:            "Jane Doe",
				Email:           "jane@gmail.com",
				Password:        "test12345",
				PasswordConfirm: "test12345",
				Roles:           []string{"user"},
			},
			expectError: true,
			statusCode:  http.StatusConflict,
		},
		"invalid roles": {
			input: users.NewUser{
				Name:            "John Doe",
				Email:           "john@gmail.com",
				Password:        "test1234",
				PasswordConfirm: "test1234",
				Roles:           []string{"someRole"},
			},
			expectError: true,
			statusCode:  http.StatusBadRequest,
		},
		"invalid inputs": {
			input: users.NewUser{
				Name:            "n",
				Email:           "www",
				Password:        "test",
				PasswordConfirm: "test123",
			},
			expectError:   true,
			statusCode:    http.StatusBadRequest,
			invalidFields: []string{"name", "email", "password", "passwordConfirm", "roles"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			v, err := errs.NewAppValidator()
			if err != nil {
				t.Fatalf("expected to create the app validator: %s", err)
			}

			userId := uuid.New()
			userRepo := memory.Repository{
				Users: map[uuid.UUID]user.User{
					userId: {
						//duplciated
						Id: userId,
						Email: mail.Address{
							Name:    "jane",
							Address: "jane@gmail.com",
						},
					},
				},
			}

			userService := user.NewService(&userRepo)

			h := users.Handler{
				Validator:    v,
				UsersService: userService,
			}

			var buff bytes.Buffer

			err = json.NewEncoder(&buff).Encode(test.input)
			if err != nil {
				t.Fatalf("expected the input to be encoded in json: %s", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/api/users", &buff)
			w := httptest.NewRecorder()

			err = h.CreateUser(context.Background(), w, req)

			if !test.expectError {
				//success
				if err != nil {
					t.Fatalf("expected the user to be created: %s", err)
				}

				if w.Result().StatusCode != test.statusCode {
					t.Errorf("w.Status= %d, got %d", http.StatusCreated, w.Result().StatusCode)
				}

				var resp users.User
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("expected response to be decoded into user: %s", err)
				}

				if resp.Email != test.input.Email {
					t.Errorf("resp.Email= %s , got %s", test.input.Email, resp.Email)
				}

				if resp.Name != test.input.Name {
					t.Errorf("resp.Name= %s, got %s", test.input.Name, resp.Name)
				}

				if !reflect.DeepEqual(resp.Roles, test.input.Roles) {
					t.Logf("got: %v", resp.Roles)
					t.Logf("want: %v", test.input.Roles)
					t.Error("expected the roles to be the same")
				}
			} else {
				//failure

				var failureResp *errs.AppError
				//we end up in error handler middleware
				if !errors.As(err, &failureResp) {
					t.Fatalf("expected the failure error to be an *appError, got %T", err)
				}

				if failureResp.Code != test.statusCode {
					t.Errorf("FailureResp.Code=%d, got %d", test.statusCode, failureResp.Code)
				}

				if failureResp.Fields != nil {
					//look for invalid field names
					for name := range failureResp.Fields {
						if !slices.Contains(test.invalidFields, name) {
							t.Errorf("expected %q field to be invalid", name)
						}
					}
				}
			}

		})
	}

}

func TestGetUserById(t *testing.T) {

	tests := map[string]struct {
		input       string
		expectError bool
		statusCode  int
	}{
		"success": {
			input:       "3dc3bbbc-811a-4bb8-a6fc-fccd709e8158",
			expectError: false,
			statusCode:  http.StatusOK,
		},
		"not found": {
			input:       "a18fe19d-797a-42f5-85f6-6cac36eae323",
			expectError: true,
			statusCode:  http.StatusNotFound,
		},
		"invalid uuid": {
			input:       "hshsgaga",
			expectError: true,
			statusCode:  http.StatusBadRequest,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			v, err := errs.NewAppValidator()
			if err != nil {
				t.Fatalf("expected to create the app validator: %s", err)
			}

			hashed, err := bcrypt.GenerateFromPassword([]byte("test1234"), bcrypt.MinCost)
			if err != nil {
				t.Fatalf("expected to generate a hash: %s", err)
			}

			now := time.Now()
			userId := uuid.MustParse("3dc3bbbc-811a-4bb8-a6fc-fccd709e8158")
			usr := user.User{
				Id: userId,
				Email: mail.Address{
					Name:    "jane",
					Address: "jane@gmail.com",
				},
				Name:         "Jane Doe",
				Roles:        []user.Role{user.RoleUser},
				PasswordHash: hashed,
				Enabled:      true,
				CreatedAt:    now,
				UpdatedAt:    now,
			}

			userRepo := memory.Repository{
				Users: map[uuid.UUID]user.User{
					userId: usr,
				},
			}

			userService := user.NewService(&userRepo)

			h := users.Handler{
				Validator:    v,
				UsersService: userService,
			}

			r := httptest.NewRequest(http.MethodGet, "/v1/api/users/"+test.input, nil)
			r.SetPathValue("id", test.input)

			w := httptest.NewRecorder()

			err = h.GetUserById(context.Background(), w, r)
			if !test.expectError {
				//success
				if err != nil {
					t.Fatalf("expected to fetch user: %s", err)
				}

				var resp users.User
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("expected to decode the user from response body: %s", err)
				}

				if resp.ID != usr.Id.String() {
					t.Errorf("resp.ID= %s, got %s", userId, resp.ID)
				}

				if resp.Name != usr.Name {
					t.Errorf("resp.Name= %s, got %s", usr.Name, resp.Name)
				}

				if resp.Email != usr.Email.Address {
					t.Errorf("resp.Email= %s, got %s", usr.Email.Address, resp.Email)
				}
			} else {
				//failure
				var appError *errs.AppError
				if !errors.As(err, &appError) {
					t.Fatalf("expected the error to be *appError: %T", err)
				}

				if appError.Code != test.statusCode {
					t.Errorf("appError.Code= %d, got %d", test.statusCode, appError.Code)
				}
			}
		})
	}

}
