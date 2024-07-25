package mid_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/errs"
	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/mid"
	"github.com/hamidoujand/task-scheduler/foundation/web"
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
			t.Parallel()
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
