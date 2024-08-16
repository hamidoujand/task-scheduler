package tasks

import (
	"errors"
	"net/http"
	"strconv"
)

func parsePagination(r *http.Request) (rows int, page int, err error) {
	deafultRows := 10
	defaultPage := 1

	rowsString := r.URL.Query().Get("rows")
	rows = deafultRows

	if rowsString != "" {
		var err error
		rows, err = strconv.Atoi(rowsString)
		if err != nil || rows <= 0 {
			return 0, 0, errors.New("invalid rows parameter")
		}
	}

	pageString := r.URL.Query().Get("page")
	page = defaultPage

	if pageString != "" {
		var err error
		page, err = strconv.Atoi(pageString)
		if err != nil || page <= 0 {
			return 0, 0, errors.New("invalid page parameter")
		}
	}
	return rows, page, nil
}
