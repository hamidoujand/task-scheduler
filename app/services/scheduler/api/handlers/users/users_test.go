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
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/auth"
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

func TestDeleteUserById(t *testing.T) {
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

	adminId := uuid.New()

	tests := map[string]struct {
		userId          string
		user            user.User
		expectError     bool
		statusCode      int
		unauthenticated bool
	}{
		"delete its own account": {
			userId:      "3dc3bbbc-811a-4bb8-a6fc-fccd709e8158",
			user:        usr,
			statusCode:  http.StatusNoContent,
			expectError: false,
		},

		"admin deleting other users": {
			userId: "3dc3bbbc-811a-4bb8-a6fc-fccd709e8158",
			user: user.User{
				Id:   adminId,
				Name: "Admin",
				Email: mail.Address{
					Name:    "Admin",
					Address: "admin@gmail.com",
				},
				Roles:        []user.Role{user.RoleAdmin},
				PasswordHash: hashed,
				Enabled:      true,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			expectError: false,
			statusCode:  http.StatusNoContent,
		},
		"random user trying to delete another user": {
			userId: "3dc3bbbc-811a-4bb8-a6fc-fccd709e8158",
			user: user.User{
				Id:   uuid.New(),
				Name: "Random",
				Email: mail.Address{
					Name:    "Random",
					Address: "random@random.com",
				},
				Roles: []user.Role{user.RoleUser},
			},
			expectError: true,
			statusCode:  http.StatusUnauthorized,
		},
		"admin deleting not found user": {
			userId: uuid.NewString(),
			user: user.User{
				Id:   adminId,
				Name: "Admin",
				Email: mail.Address{
					Name:    "Admin",
					Address: "admin@gmail.com",
				},
				Roles:        []user.Role{user.RoleAdmin},
				PasswordHash: hashed,
				Enabled:      true,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			expectError: true,
			statusCode:  http.StatusNotFound,
		},
		"unauthenticated user trying to delelte": {
			userId:          "3dc3bbbc-811a-4bb8-a6fc-fccd709e8158",
			expectError:     true,
			statusCode:      http.StatusUnauthorized,
			unauthenticated: true,
		},
		"invalid uuid": {
			userId:      "sksjsshsh",
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

			userRepo := memory.Repository{
				Users: map[uuid.UUID]user.User{},
			}
			//add user
			userRepo.Create(context.Background(), usr)

			userService := user.NewService(&userRepo)

			h := users.Handler{
				Validator:    v,
				UsersService: userService,
			}

			req := httptest.NewRequest(http.MethodDelete, "/v1/api/users/"+test.userId, nil)
			w := httptest.NewRecorder()
			req.SetPathValue("id", test.userId)

			//add auth user into ctx
			var ctx context.Context
			if !test.unauthenticated {
				ctx = auth.SetUser(context.Background(), test.user)
			} else {
				ctx = context.Background()
			}

			err = h.DeleteUserById(ctx, w, req)
			if !test.expectError {
				//success path
				if err != nil {
					t.Fatalf("expected the user to be deleteed: %s", err)
				}

				if w.Result().StatusCode != http.StatusNoContent {
					t.Errorf("w.StatusCode = %d, got %d", http.StatusNoContent, w.Result().StatusCode)
				}
			} else {
				//failure path
				var appErr *errs.AppError
				if !errors.As(err, &appErr) {
					t.Fatalf("expected the error type to be *appError, got %T", err)
				}

				if appErr.Code != test.statusCode {
					t.Errorf("appErr.Code=%d, got %d", test.statusCode, appErr.Code)
				}
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	//Setup

	hashed, err := bcrypt.GenerateFromPassword([]byte("test1234"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("expected to hash the password: %s", err)
	}

	v, err := errs.NewAppValidator()
	if err != nil {
		t.Fatalf("expected to create the app validator: %s", err)
	}

	userId := uuid.New()
	created := time.Now().Add(-time.Hour)

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
		CreatedAt:    created,
		UpdatedAt:    created,
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

	//==========================================================================
	type input struct {
		Name            *string `json:"name,omitempty"`
		Email           *string `json:"email,omitempty"`
		Enabled         *bool   `json:"enabled,omitempty"`
		Password        *string `json:"password,omitempty"`
		PasswordConfirm *string `json:"passwordConfirm,omitempty"`
	}
	tests := map[string]struct {
		input       input
		userId      uuid.UUID
		usr         user.User
		expectError bool
		statusCode  int
		fields      []string
	}{
		"user can update its own account": {
			input: input{
				Name:            stringPointer("John Doe"),
				Email:           stringPointer("john@gmail.com"),
				Enabled:         boolPointer(false),
				Password:        stringPointer("test54321"),
				PasswordConfirm: stringPointer("test54321"),
			},
			userId:      userId,
			usr:         usr,
			expectError: false,
			statusCode:  http.StatusOK,
		},

		"admin can update anyone": {
			input: input{
				Name:            stringPointer("John Doe"),
				Email:           stringPointer("john@gmail.com"),
				Enabled:         boolPointer(false),
				Password:        stringPointer("test54321"),
				PasswordConfirm: stringPointer("test54321"),
			},
			userId: userId,
			usr: user.User{
				Id:   uuid.New(),
				Name: "Admin",
				Email: mail.Address{
					Name:    "Admin",
					Address: "admin@gmail.com",
				},
				Roles:        []user.Role{user.RoleAdmin},
				PasswordHash: []byte("pass"),
				Enabled:      false,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			expectError: false,
			statusCode:  http.StatusOK,
		},
		"random people can not update others": {
			input:  input{},
			userId: userId,
			usr: user.User{
				Roles: []user.Role{user.RoleUser},
			},
			expectError: true,
			statusCode:  http.StatusUnauthorized,
		},

		"admin updating user that not exists": {
			input: input{
				Name:            stringPointer("John Doe"),
				Email:           stringPointer("john@gmail.com"),
				Enabled:         boolPointer(false),
				Password:        stringPointer("test54321"),
				PasswordConfirm: stringPointer("test54321"),
			},
			userId: uuid.New(),
			usr: user.User{
				Roles: []user.Role{user.RoleAdmin},
			},
			expectError: true,
			statusCode:  http.StatusNotFound,
		},

		"invalid inputs": {
			input: input{
				Name:            stringPointer("w"),
				Email:           stringPointer("www"),
				Password:        stringPointer("pa"),
				PasswordConfirm: stringPointer("pas"),
			},
			userId:      userId,
			usr:         usr,
			expectError: true,
			statusCode:  http.StatusBadRequest,
			fields:      []string{"name", "email", "password", "passwordConfirm"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var buff bytes.Buffer
			if json.NewEncoder(&buff).Encode(test.input); err != nil {
				t.Fatalf("expected to marshal the input: %s", err)
			}
			path := "/v1/api/users/" + test.userId.String()

			r := httptest.NewRequest(http.MethodPut, path, &buff)
			r.SetPathValue("id", test.userId.String())
			w := httptest.NewRecorder()

			ctx := auth.SetUser(r.Context(), test.usr)

			err := h.UpdateUser(ctx, w, r)
			if !test.expectError {
				if err != nil {
					t.Errorf("expected the user to be updated: %s", err)
				}

				statusCode := w.Result().StatusCode
				if statusCode != http.StatusOK {
					t.Errorf("statusCode= %d, got %d", http.StatusOK, statusCode)
				}
				var successResp users.User
				if err := json.NewDecoder(w.Body).Decode(&successResp); err != nil {
					t.Fatalf("expected to decode the response body: %s", err)
				}
				if test.input.Email != nil {
					if successResp.Email != *test.input.Email {
						t.Errorf("updated.Email=%s, got %s", *test.input.Email, successResp.Email)
					}
				}

				if test.input.Name != nil {
					if successResp.Name != *test.input.Name {
						t.Errorf("updated.Name=%s, got %s", *test.input.Name, successResp.Name)
					}
				}
				if test.input.Enabled != nil {
					if successResp.Enabled != *test.input.Enabled {
						t.Errorf("updated.Enabled=%t, got %t", *test.input.Enabled, successResp.Enabled)
					}
				}

				//you do not have access to password from response
				if test.input.Password != nil {
					//access the user from repo directly
					fetched, err := userService.GetUserById(context.Background(), uuid.MustParse(successResp.ID))
					if err != nil {
						t.Fatalf("expected to fetch user: %s", err)
					}

					if err := bcrypt.CompareHashAndPassword(fetched.PasswordHash, []byte(*test.input.Password)); err != nil {
						t.Errorf("passwords does not match")
					}
				}

			} else {
				//failure path
				var appErr *errs.AppError
				if !errors.As(err, &appErr) {
					t.Fatalf("expected error to be of type *appError, got: %T", err)
				}
				if appErr.Code != test.statusCode {
					t.Errorf("status= %d, got %d", test.statusCode, appErr.Code)
				}

				if appErr.Fields != nil {
					for name := range appErr.Fields {
						if !slices.Contains(test.fields, name) {
							t.Errorf("expected field %s to be invalid", name)
						}
					}
				}
			}
		})
	}

}

func TestUpdateRoles(t *testing.T) {
	//setup
	hashed, err := bcrypt.GenerateFromPassword([]byte("test1234"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("expected to hash the password: %s", err)
	}

	v, err := errs.NewAppValidator()
	if err != nil {
		t.Fatalf("expected to create the app validator: %s", err)
	}

	userId := uuid.New()
	adminId := uuid.New()
	created := time.Now().Add(-time.Hour)

	normal := user.User{
		Id: userId,
		Email: mail.Address{
			Name:    "jane",
			Address: "jane@gmail.com",
		},
		Name:         "Jane Doe",
		Roles:        []user.Role{user.RoleUser},
		PasswordHash: hashed,
		Enabled:      true,
		CreatedAt:    created,
		UpdatedAt:    created,
	}
	admin := user.User{
		Id:   adminId,
		Name: "Admin",
		Email: mail.Address{
			Name:    "Admin",
			Address: "admin@gmail.com",
		},
		Roles:        []user.Role{user.RoleAdmin},
		PasswordHash: hashed,
		Enabled:      true,
		CreatedAt:    created,
		UpdatedAt:    created,
	}

	userRepo := memory.Repository{
		Users: map[uuid.UUID]user.User{
			userId:  normal,
			adminId: admin,
		},
	}

	userService := user.NewService(&userRepo)

	h := users.Handler{
		Validator:    v,
		UsersService: userService,
	}

	tests := map[string]struct {
		input       []string
		userId      uuid.UUID
		updater     user.User
		expectError bool
		statusCode  int
	}{
		"sucess": {
			input:       []string{"admin", "user"},
			userId:      userId,
			updater:     admin,
			expectError: false,
			statusCode:  http.StatusOK,
		},

		"random user updating roles": {
			input:       []string{"admin"},
			userId:      userId,
			updater:     normal,
			expectError: true,
			statusCode:  http.StatusUnauthorized,
		},
		"admin updating a not found user": {
			input:       []string{"user"},
			userId:      uuid.New(),
			updater:     admin,
			expectError: true,
			statusCode:  http.StatusNotFound,
		},
		"admin passing invalid roles": {
			input:       []string{"ceo"},
			userId:      userId,
			updater:     admin,
			expectError: true,
			statusCode:  http.StatusBadRequest,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ur := users.UpdateRole{
				Roles: test.input,
			}
			var buff bytes.Buffer
			if err := json.NewEncoder(&buff).Encode(ur); err != nil {
				t.Fatalf("expected to marshal roles: %s", err)
			}

			path := "/v1/api/users/" + test.userId.String()
			r := httptest.NewRequest(http.MethodPut, path, &buff)
			r.SetPathValue("id", test.userId.String())

			w := httptest.NewRecorder()

			ctx := auth.SetUser(context.Background(), test.updater)

			err = h.UpdateRole(ctx, w, r)

			if !test.expectError {
				if err != nil {
					t.Fatalf("expected to update the roles: %s", err)
				}

				statusCode := w.Result().StatusCode
				if statusCode != test.statusCode {
					t.Fatalf("statusCode= %d, got %d", http.StatusOK, statusCode)
				}

				var successResp users.User
				if err := json.NewDecoder(w.Body).Decode(&successResp); err != nil {
					t.Fatalf("expected to decode the response: %s", err)
				}
				for _, role := range ur.Roles {
					if !slices.Contains(successResp.Roles, role) {
						t.Errorf("expected role %q to in roles", role)
					}
				}

			} else {

				var appErr *errs.AppError
				if !errors.As(err, &appErr) {
					t.Fatalf("expected the error type to be *appError, got: %T", err)
				}

				if appErr.Code != test.statusCode {
					t.Errorf("status= %d, got %d", test.statusCode, appErr.Code)
				}
			}

		})
	}

}

const (
	kid = "s4sKIjD9kIRjxs2tulPqGLdxSfgPErRN1Mu3Hd9k9NQ"
)

func TestSignup(t *testing.T) {
	//setup
	hashed, err := bcrypt.GenerateFromPassword([]byte("test1234"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("expected to hash the password: %s", err)
	}

	v, err := errs.NewAppValidator()
	if err != nil {
		t.Fatalf("expected to create the app validator: %s", err)
	}

	userId := uuid.New()
	created := time.Now().Add(-time.Hour)

	random := user.User{
		Id: userId,
		Email: mail.Address{
			Name:    "jane",
			Address: "jane@gmail.com",
		},
		Name:         "Jane Doe",
		Roles:        []user.Role{user.RoleUser},
		PasswordHash: hashed,
		Enabled:      true,
		CreatedAt:    created,
		UpdatedAt:    created,
	}
	userRepo := memory.Repository{
		Users: map[uuid.UUID]user.User{
			userId: random,
		},
	}
	ks := auth.NewMockKeyStore(t)
	userService := user.NewService(&userRepo)
	a := auth.New(ks, userService)
	tokenExpiresAt := time.Now().Add(time.Hour)

	h := users.Handler{
		Validator:      v,
		UsersService:   userService,
		Auth:           a,
		ActiveKID:      kid,
		TokenExpiresAt: tokenExpiresAt,
	}

	tests := map[string]struct {
		input       users.SignUp
		expectError bool
		statusCode  int
		fields      []string
	}{
		"success": {
			input: users.SignUp{
				Name:            "John Doe",
				Email:           "john@gmail.com",
				Password:        "test1234",
				PasswordConfirm: "test1234",
			},
			expectError: false,
			statusCode:  http.StatusCreated,
		},
		"using duplicated email": {
			input: users.SignUp{
				Name:            "Jane Doe",
				Email:           "jane@gmail.com",
				Password:        "test1234",
				PasswordConfirm: "test1234",
			},
			expectError: true,
			statusCode:  http.StatusConflict,
		},
		"invalid input": {
			input: users.SignUp{
				Name:            "ss",
				Email:           "sa",
				Password:        "sa",
				PasswordConfirm: "sas",
			},
			expectError: true,
			statusCode:  http.StatusBadRequest,
			fields:      []string{"name", "email", "password", "passwordConfirm"},
		},
	}

	for name, test := range tests {

		t.Run(name, func(t *testing.T) {

			var buff bytes.Buffer
			if err := json.NewEncoder(&buff).Encode(test.input); err != nil {
				t.Fatalf("expected the input to be encoded to json: %s", err)
			}

			r := httptest.NewRequest(http.MethodPost, "/v1/api/users/signup", &buff)
			w := httptest.NewRecorder()

			err = h.Signup(r.Context(), w, r)
			if !test.expectError {
				if err != nil {
					t.Fatalf("expected the user to signup: %s", err)
				}

				statusCode := w.Result().StatusCode
				if statusCode != test.statusCode {
					t.Fatalf("status= %d, got %d", test.statusCode, statusCode)
				}
				var successResp users.User
				if err := json.NewDecoder(w.Body).Decode(&successResp); err != nil {
					t.Fatalf("expected to decode the response body: %s", err)
				}
				if successResp.Token == "" {
					t.Errorf("token= %s, got %s", "<jwt>", successResp.Token)
				}

				bearer := "Bearer " + successResp.Token

				usr, err := h.Auth.ValidateToken(context.Background(), bearer)
				if err != nil {
					t.Fatalf("expected the token to be valid: %s", err)
				}

				if usr.Id.String() != successResp.ID {
					t.Errorf("user.Id= %s, got %s", successResp.ID, usr.Id.String())
				}
			} else {
				var appErr *errs.AppError
				if !errors.As(err, &appErr) {
					t.Fatalf("expected the error type to be *appError, got: %T", err)
				}

				if appErr.Code != test.statusCode {
					t.Errorf("status= %d, got %d", test.statusCode, appErr.Code)
				}

				if appErr.Fields != nil {
					for name := range appErr.Fields {
						if !slices.Contains(test.fields, name) {
							t.Errorf("expected field %s to be invalid", name)
						}
					}
				}
			}
		})

	}

}

func TestLogin(t *testing.T) {
	//setup
	rawPass := "test1234"
	hashed, err := bcrypt.GenerateFromPassword([]byte(rawPass), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("expected to hash the password: %s", err)
	}

	v, err := errs.NewAppValidator()
	if err != nil {
		t.Fatalf("expected to create the app validator: %s", err)
	}

	userId := uuid.New()
	created := time.Now().Add(-time.Hour)

	random := user.User{
		Id: userId,
		Email: mail.Address{
			Name:    "jane",
			Address: "jane@gmail.com",
		},
		Name:         "Jane Doe",
		Roles:        []user.Role{user.RoleUser},
		PasswordHash: hashed,
		Enabled:      true,
		CreatedAt:    created,
		UpdatedAt:    created,
	}
	userRepo := memory.Repository{
		Users: map[uuid.UUID]user.User{
			userId: random,
		},
	}
	ks := auth.NewMockKeyStore(t)
	userService := user.NewService(&userRepo)
	a := auth.New(ks, userService)
	tokenExpiresAt := time.Now().Add(time.Hour)

	h := users.Handler{
		Validator:      v,
		UsersService:   userService,
		Auth:           a,
		ActiveKID:      kid,
		TokenExpiresAt: tokenExpiresAt,
	}

	tests := map[string]struct {
		input       users.Login
		statusCode  int
		expectError bool
		fields      []string
	}{
		"success": {
			input: users.Login{
				Email:    "jane@gmail.com",
				Password: rawPass,
			},
			statusCode:  http.StatusOK,
			expectError: false,
		},
		"invalid input": {
			input: users.Login{
				Email:    "email",
				Password: "pass",
			},
			statusCode:  http.StatusBadRequest,
			expectError: true,
			fields:      []string{"email", "password"},
		},

		"user not found": {
			input: users.Login{
				Email:    "john@gmail.com",
				Password: "test1234",
			},
			statusCode:  http.StatusBadRequest,
			expectError: true,
		},
		"wrong password": {
			input: users.Login{
				Email:    "jane@gmail.com",
				Password: "test1234567",
			},
			statusCode:  http.StatusBadRequest,
			expectError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			var buff bytes.Buffer
			if err := json.NewEncoder(&buff).Encode(test.input); err != nil {
				t.Fatalf("expected the input to be encoded into json: %s", err)
			}

			r := httptest.NewRequest(http.MethodPost, "/v1/api/users/login", &buff)
			w := httptest.NewRecorder()

			err = h.Login(context.Background(), w, r)
			if !test.expectError {

				if err != nil {
					t.Fatalf("expected to login, got err: %s", err)
				}

				statusCode := w.Result().StatusCode
				if statusCode != test.statusCode {
					t.Fatalf("status= %d, got %d", test.statusCode, statusCode)
				}

				var successResp users.User
				if err := json.NewDecoder(w.Body).Decode(&successResp); err != nil {
					t.Fatalf("expected to decode the response body: %s", err)
				}
				if successResp.Token == "" {
					t.Errorf("token= %s, got %s", "<jwt>", successResp.Token)
				}

				bearer := "Bearer " + successResp.Token

				usr, err := h.Auth.ValidateToken(context.Background(), bearer)
				if err != nil {
					t.Fatalf("expected the token to be valid: %s", err)
				}

				if usr.Id.String() != successResp.ID {
					t.Errorf("user.Id= %s, got %s", successResp.ID, usr.Id.String())
				}
			} else {
				var appErr *errs.AppError
				if !errors.As(err, &appErr) {
					t.Fatalf("expected the error type to be *appError, got: %T", err)
				}

				if appErr.Code != test.statusCode {
					t.Errorf("status= %d, got %d", test.statusCode, appErr.Code)
				}

				if appErr.Fields != nil {
					for name := range appErr.Fields {
						if !slices.Contains(test.fields, name) {
							t.Errorf("expected field %s to be invalid", name)
						}
					}
				}
			}
		})
	}

}

func stringPointer(str string) *string { return &str }
func boolPointer(b bool) *bool         { return &b }
