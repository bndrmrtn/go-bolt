package bolt

import (
	"errors"
	"regexp"
	"strconv"

	"github.com/google/uuid"
)

func registerDefaultRouteValidators(router RouterParamValidator) {
	router.RegisterRouteParamValidator("int", validateInt)
	router.RegisterRouteParamValidator("bool", validateBool)
	router.RegisterRouteParamValidator("uuid", validateUUIDv4)
	router.RegisterRouteParamValidator("alpha", validateAlpha)
	router.RegisterRouteParamValidator("alphanumeric", validateAlphaNumeric)
}

func validateInt(value string) (string, error) {
	_, err := strconv.Atoi(value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func validateBool(value string) (string, error) {
	_, err := strconv.ParseBool(value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func validateUUIDv4(value string) (string, error) {
	_, err := uuid.Parse(value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func validateAlpha(value string) (string, error) {
	ok := regexp.MustCompile("^[a-zA-Z]+$").MatchString(value)
	if !ok {
		return "", errors.New("param is not alpha")
	}
	return value, nil
}

func validateAlphaNumeric(value string) (string, error) {
	ok := regexp.MustCompile("^[a-zA-Z0-9]+$").MatchString(value)
	if !ok {
		return "", errors.New("param is not alphanumeric")
	}
	return value, nil
}
