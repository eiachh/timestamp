package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var Reader *TimestampReader

func main() {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard
	r := setupServer()
	go startServer(r)

	client := &http.Client{}
	reqPOST, _ := http.NewRequest(http.MethodPost, "http://localhost:8080/timestamp", bytes.NewBufferString("1740863149"))
	reqPOST.Header.Set("Content-Type", "text/plain")

	postResp, postErr := client.Do(reqPOST)
	if postErr != nil {
		fmt.Println(postErr)
	}
	defer postResp.Body.Close()

	reqGET, _ := http.NewRequest(http.MethodGet, "http://localhost:8080/timestamp", nil)
	reqGET.Header.Set("Content-Type", "text/plain")
	getResp, getErr := client.Do(reqGET)
	if getErr != nil {
		fmt.Println(getErr)
	}
	defer getResp.Body.Close()

	body, _ := io.ReadAll(getResp.Body)
	fmt.Println(string(body))

}
func setupServer() *gin.Engine {
	Reader = NewTimestampReader()
	Reader.Start()

	r := gin.Default()
	r.Use(EnforcePlainText())

	r.GET("/timestamp", GetTimestamp)
	r.POST("/timestamp", SetTimestamp)

	return r
}

func startServer(r *gin.Engine) {
	r.Run(":8080")
}

type TimestampReader struct {
	timestamp   *time.Time
	timestampCh chan time.Time
	quitCh      chan struct{}
	unlockCh    chan struct{}
	lockCh      chan struct{}
}

func NewTimestampReader() *TimestampReader {
	return &TimestampReader{
		timestamp:   &time.Time{},
		timestampCh: make(chan time.Time),
		quitCh:      make(chan struct{}),

		unlockCh: make(chan struct{}),
		lockCh:   make(chan struct{}),
	}
}
func (reader *TimestampReader) Start() {
	go reader.loop()
	go reader.notAMutex()
}

func (reader *TimestampReader) loop() {
	for {
		select {
		case timestamp := <-reader.timestampCh:
			reader.setTimestamp(timestamp)
		case <-reader.quitCh:
			return
		}
	}
}

func (reader *TimestampReader) notAMutex() {
	for {
		<-reader.lockCh
		<-reader.unlockCh
	}
}

func (reader *TimestampReader) getTimestamp() time.Time {
	reader.lockCh <- struct{}{}
	defer func() { reader.unlockCh <- struct{}{} }()
	copiedTimestamp := *reader.timestamp
	return copiedTimestamp
}

func (reader *TimestampReader) setTimestamp(timestamp time.Time) {
	reader.lockCh <- struct{}{}
	defer func() { reader.unlockCh <- struct{}{} }()
	reader.timestamp = &timestamp
}

func EnforcePlainText() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.ContentType() != "text/plain" {
			c.AbortWithStatus(http.StatusUnsupportedMediaType)
			c.String(http.StatusUnsupportedMediaType, "Only 'text/plain' content type is allowed")
			return
		}
		c.Next()
	}
}

func GetTimestamp(c *gin.Context) {
	timestamp := strconv.FormatInt(Reader.getTimestamp().Unix(), 10)
	c.String(http.StatusOK, timestamp)
}

func SetTimestamp(c *gin.Context) {
	if c.Request.Body == nil {
		c.String(http.StatusBadRequest, "body cannot be nil or empty")
		return
	}
	timestampStr, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "failed to read the body")
		return
	}

	time, convertErr := TimeFromUnixTimeString(string(timestampStr))
	if convertErr != nil {
		c.String(http.StatusBadRequest, "failed to convert body to unix time")
		return
	}
	Reader.timestampCh <- time

	c.String(http.StatusOK, "OK")
}

func TimeFromUnixTimeString(unixTime string) (time.Time, error) {
	timestampInt64, err := strconv.ParseInt(strings.TrimSpace(unixTime), 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(timestampInt64, 0), nil
}
