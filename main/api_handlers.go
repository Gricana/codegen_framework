package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

func (h *MyApi) handlerProfile(w http.ResponseWriter, r *http.Request) {
	var params ProfileParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("invalid request body")})
		return
	}
	result, err := h.Profile(context.Background(), params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusInternalServerError, Err: err})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *MyApi) handlerCreate(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "100500" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusUnauthorized, Err: fmt.Errorf("unauthorized")})
		return
	}
	var params CreateParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("invalid request body")})
		return
	}
	if len(params.Login) < 10 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("Login length must be >= 10")})
		return
	}
	switch params.Status {
	case "user":
	case "moderator":
	case "admin":
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("Status must be one of [user|moderator|admin]")})
		return
	}
	if params.Age < 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("Age must be >= 0")})
		return
	}
	if params.Age > 128 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("Age must be <= 128")})
		return
	}
	result, err := h.Create(context.Background(), params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusInternalServerError, Err: err})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *OtherApi) handlerCreate(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "100500" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusUnauthorized, Err: fmt.Errorf("unauthorized")})
		return
	}
	var params OtherCreateParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("invalid request body")})
		return
	}
	if len(params.Username) < 3 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("Username length must be >= 3")})
		return
	}
	switch params.Class {
	case "warrior":
	case "sorcerer":
	case "rouge":
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("Class must be one of [warrior|sorcerer|rouge]")})
		return
	}
	if params.Level < 1 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("Level must be >= 1")})
		return
	}
	if params.Level > 50 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("Level must be <= 50")})
		return
	}
	result, err := h.Create(context.Background(), params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusInternalServerError, Err: err})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "" && r.URL.Path == "/user/profile" {
		h.handlerProfile(w, r)
		return
	}
	if r.Method == "POST" && r.URL.Path == "/user/create" {
		h.handlerCreate(w, r)
		return
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusNotFound, Err: fmt.Errorf("unknown method %s on %s", r.Method, r.URL.Path)})

}
func (h *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" && r.URL.Path == "/user/create" {
		h.handlerCreate(w, r)
		return
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusNotFound, Err: fmt.Errorf("unknown method %s on %s", r.Method, r.URL.Path)})

}
