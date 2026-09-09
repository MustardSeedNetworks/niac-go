package api

import (
	"errors"
	"net/http"
	"strconv"
)

func historyCursor(r *http.Request) (uint64, error) {
	value := r.URL.Query().Get("before")
	if value == "" {
		return 0, nil
	}
	before, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}
	if before == 0 {
		return 0, errors.New("zero history cursor")
	}
	return before, nil
}
