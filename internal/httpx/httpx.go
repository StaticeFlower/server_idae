// Package httpx รวม helper สำหรับอ่าน/เขียน JSON
package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

const maxBodyBytes = 1 << 20 // 1 MiB

type errorBody struct {
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "error", err)
	}
}

// Error ตอบกลับในรูป {"message": "..."} ซึ่งหน้าเว็บอ่านไปแสดงผลได้ตรง ๆ
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, errorBody{Message: message})
}

// Decode อ่าน JSON body (จำกัดขนาด 1 MiB) และต้องมีค่า JSON เพียงค่าเดียว
func Decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("unexpected data after JSON body")
	}
	return nil
}
