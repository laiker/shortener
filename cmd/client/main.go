package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func main() {
	endpoint := "http://localhost:8080/"

	fmt.Println("Type long URL")
	reader := bufio.NewReader(os.Stdin)
	long, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	long = strings.TrimSuffix(long, "\n")

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(long))
	_ = zw.Close()

	client := &http.Client{}

	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(buf.String()))
	if err != nil {
		panic(err)
	}

	request.Header.Add("Content-Type", "text/plain")
	request.Header.Set("Content-Encoding", "gzip")
	request.Header.Set("Accept-Encoding", "gzip")

	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}

	fmt.Println("Status ", response.Status)
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	if err != nil {
		panic(err)
	}

	gzip, err := gzip.NewReader(bytes.NewReader(body))

	if err != nil {
		panic(err)
	}

	body, err = io.ReadAll(gzip)

	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}
