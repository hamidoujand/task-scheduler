package errs

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

// AppValidator represents the validator used for model validation
type AppValidator struct {
	validate   *validator.Validate
	translator ut.Translator
}

// NewAppValidator creates and setup a validator and a translator
func NewAppValidator() (*AppValidator, error) {
	v := validator.New(validator.WithRequiredStructEnabled())

	//english translator
	translator, _ := ut.New(en.New(), en.New()).GetTranslator("en")

	err := en_translations.RegisterDefaultTranslations(v, translator)
	if err != nil {
		return nil, fmt.Errorf("registering default translator: %w", err)
	}

	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), " ", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	//register custom validators
	v.RegisterValidation("commonCommands", commonCommands)
	v.RegisterValidation("commonArgs", validCommandArgs)
	v.RegisterValidation("validScheduledAt", validScheduledAt)

	return &AppValidator{
		validate:   v,
		translator: translator,
	}, nil
}

// Check is going to validate a struct and then in case validation failed, returns an error of type *AppError.
func (av *AppValidator) Check(val any) (map[string]string, bool) {
	err := av.validate.Struct(val)

	if err != nil {
		//check failed
		var vErrs validator.ValidationErrors

		if !errors.As(err, &vErrs) {
			//return raw err
			return nil, false
		}

		customValidatorsErrMsg := map[string]string{
			"Command.commonCommands":       "command is not supported in this system",
			"Args.commonArgs":              "provided args contains invalid chars",
			"ScheduledAt.validScheduledAt": "scheduledAt most be greater or equal to current time",
		}

		fields := make(map[string]string, len(vErrs))

		for _, vErr := range vErrs {
			fieldName := fmt.Sprintf("%s.%s", vErr.StructField(), vErr.Tag())
			msg, ok := customValidatorsErrMsg[fieldName]
			if ok {
				fields[vErr.Field()] = msg
			} else {
				fields[vErr.Field()] = vErr.Translate(av.translator)
			}
		}
		return fields, false
	}
	//check succeeded
	return nil, true
}

//==============================================================================
// Custom Validators

func commonCommands(field validator.FieldLevel) bool {
	command := field.Field().String()

	//dangerous commands
	disallowed := []string{"rm", "shutdown", "format"}

	return !slices.Contains(disallowed, command)
}

func validCommandArgs(field validator.FieldLevel) bool {
	args := field.Field().Interface().([]string)
	return !slices.Contains(args, ";")
}

func validScheduledAt(fl validator.FieldLevel) bool {
	scheduledAt, ok := fl.Field().Interface().(time.Time)
	if !ok {
		return false
	}
	now := time.Now()
	return scheduledAt.After(now) || scheduledAt.Equal(now)
}
