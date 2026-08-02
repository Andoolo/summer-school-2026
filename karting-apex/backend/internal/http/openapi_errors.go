package httpapi

import "net/http"

func OpenAPIErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	WriteError(w, http.StatusBadRequest, CodeBadRequest, "Неверные параметры запроса. Проверьте корректность переданных значений.", nil)
}
