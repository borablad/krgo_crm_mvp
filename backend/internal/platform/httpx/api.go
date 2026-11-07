package httpx

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/danielgtaylor/huma/v2/validation"
)

// SessionCookie — имя cookie с токеном сессии.
const SessionCookie = "cargo_session"

// Public помечает операцию, доступную без входа (например, сам вход).
func Public(op huma.Operation) huma.Operation {
	if op.Metadata == nil {
		op.Metadata = map[string]any{}
	}
	op.Metadata["public"] = true
	return op
}

// IsPublic сообщает, можно ли вызывать операцию без сессии.
func IsPublic(op *huma.Operation) bool {
	if op == nil || op.Metadata == nil {
		return false
	}
	v, _ := op.Metadata["public"].(bool)
	return v
}

// NewAPI создаёт Huma API поверх стандартного mux. OpenAPI доступен по /openapi.json, документация — по /docs.
func NewAPI(mux *http.ServeMux, version string) huma.API {
	cfg := huma.DefaultConfig("Cargo CRM API", version)
	cfg.OpenAPIPath = "/openapi"
	cfg.DocsPath = "/docs"
	cfg.Info.Description = "Внутренняя система: CRM, казначейство, склад."
	cfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"session": {Type: "apiKey", In: "cookie", Name: SessionCookie},
	}
	cfg.Security = []map[string][]string{{"session": {}}}
	return humago.New(mux, cfg)
}

// FieldError — ошибка конкретного поля тела запроса; фронтенд показывает её под полем.
func FieldError(field, message string) *huma.ErrorDetail {
	return &huma.ErrorDetail{Location: "body." + field, Message: message}
}

// Invalid — ответ 422 с общим текстом и ошибками полей.
func Invalid(detail string, fields ...*huma.ErrorDetail) error {
	errs := make([]error, 0, len(fields))
	for _, f := range fields {
		errs = append(errs, f)
	}
	return huma.Error422UnprocessableEntity(detail, errs...)
}

// Сообщения валидации Huma на русском. Это запасной уровень: основные проверки
// с человеческими текстами делают сервисы и фронтенд.
func init() {
	// Списки в ответах всегда непустые срезы (make(..., 0)), поэтому в схеме они не nullable:
	// иначе фронтенд получает тип `T[] | null` на каждый список.
	huma.DefaultArrayNullable = false

	validation.MsgExpectedRequiredProperty = "заполните поле %s"
	validation.MsgUnexpectedProperty = "неизвестное поле"
	validation.MsgExpectedString = "ожидается текст"
	validation.MsgExpectedNumber = "ожидается число"
	validation.MsgExpectedInteger = "ожидается целое число"
	validation.MsgExpectedBoolean = "ожидается да или нет"
	validation.MsgExpectedArray = "ожидается список"
	validation.MsgExpectedObject = "ожидается объект"
	validation.MsgExpectedMinLength = "не короче %d символов"
	validation.MsgExpectedMaxLength = "не длиннее %d символов"
	validation.MsgExpectedMinimumNumber = "не меньше %v"
	validation.MsgExpectedExclusiveMinimumNumber = "больше %v"
	validation.MsgExpectedMaximumNumber = "не больше %v"
	validation.MsgExpectedExclusiveMaximumNumber = "меньше %v"
	validation.MsgExpectedOneOf = "допустимые значения: %s"
	validation.MsgExpectedMatchPattern = "неверный формат (ожидается %s)"
	validation.MsgExpectedMinItems = "в списке не меньше %d элементов"
	validation.MsgExpectedMaxItems = "в списке не больше %d элементов"
}
