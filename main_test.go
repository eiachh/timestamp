package main

import (
	"bytes"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

var router *gin.Engine

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	router = setupServer()
	go startServer(router)

	exitCode := m.Run()
	os.Exit(exitCode)
}

func PerformPostTimestamp(router *gin.Engine, data string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPost, "/timestamp", bytes.NewBufferString(data))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

func PerformPostWithNilData(router *gin.Engine) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPost, "/timestamp", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

func PerformGetTimestamp(router *gin.Engine) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodGet, "/timestamp", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

func TestGetTimestampDefault(t *testing.T) {
	resp := PerformGetTimestamp(router)

	if resp.Code != http.StatusOK {
		fmt.Print("200")
	}
	respBody := resp.Body.String()
	if respBody != "-62135596800" {
		t.Errorf("Unexpected body! Expected: %s, Got: %s", "-62135596800", respBody)
	}
}

func TestGetTimestampWhenSet(t *testing.T) {
	setTime := "1740863149"
	PerformPostTimestamp(router, setTime)

	time.Sleep(time.Millisecond)

	resp := PerformGetTimestamp(router)
	if resp.Code != http.StatusOK {
		fmt.Print("200")
	}
	respBody := resp.Body.String()
	if respBody != setTime {
		t.Errorf("Unexpected body! Expected: %s, Got: %s", setTime, respBody)
	}
}

func TestSetTimestamp(t *testing.T) {
	biggerThanInt64 := strconv.FormatInt(math.MaxInt64, 10) + "1"

	tests := []struct {
		input        string
		expectedCode int
		expectedBody string
	}{
		{"1740957796", 200, "OK"},
		{"1", 200, "OK"},
		{biggerThanInt64, 400, "failed to convert body to unix time"},
		{"", 400, "failed to convert body to unix time"},
		{"1740--7796", 400, "failed to convert body to unix time"},
	}

	for _, tt := range tests {
		resp := PerformPostTimestamp(router, tt.input)

		respBody := resp.Body.String()
		if respBody != tt.expectedBody {
			t.Errorf("Unexpected body! Expected: %s, Got: %s", tt.expectedBody, respBody)
		}
		if resp.Code != tt.expectedCode {
			t.Errorf("Unexpected statuscode! Expected: %d, Got: %d", tt.expectedCode, resp.Code)
		}
	}

}

func TestSetTimestampNilBody(t *testing.T) {
	resp := PerformPostWithNilData(router)

	respBody := resp.Body.String()
	expectedRespBody := "body cannot be nil or empty"
	if respBody != expectedRespBody {
		t.Errorf("Unexpected body! Expected: %s, Got: %s", expectedRespBody, respBody)
	}
	if resp.Code != http.StatusBadRequest {
		t.Errorf("Unexpected statuscode! Expected: %d, Got: %d", http.StatusBadRequest, resp.Code)
	}
}

func TestEnforcePlainTextPOST(t *testing.T) {
	reqPOST, _ := http.NewRequest(http.MethodPost, "/timestamp", bytes.NewBufferString("1740863149"))
	reqPOST.Header.Set("Content-Type", "application/json")
	wPOST := httptest.NewRecorder()
	router.ServeHTTP(wPOST, reqPOST)

	if wPOST.Code != http.StatusUnsupportedMediaType {
		t.Errorf("Expected return code: %d, Got: %d", http.StatusUnsupportedMediaType, wPOST.Code)
	}
}

func TestEnforcePlainTextGET(t *testing.T) {
	reqGET, _ := http.NewRequest(http.MethodGet, "/timestamp", nil)
	reqGET.Header.Set("Content-Type", "application/json")
	wGET := httptest.NewRecorder()
	router.ServeHTTP(wGET, reqGET)

	if wGET.Code != http.StatusUnsupportedMediaType {
		t.Errorf("Expected return code: %d, Got: %d", http.StatusUnsupportedMediaType, wGET.Code)
	}
}
